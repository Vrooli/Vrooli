export const auth_switchCurrentAccount = {
  "fieldName": "switchCurrentAccount",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "switchCurrentAccount",
        "loc": {
          "start": 347,
          "end": 367
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 368,
              "end": 373
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 376,
                "end": 381
              }
            },
            "loc": {
              "start": 375,
              "end": 381
            }
          },
          "loc": {
            "start": 368,
            "end": 381
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLoggedIn",
              "loc": {
                "start": 389,
                "end": 399
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 389,
              "end": 399
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 404,
                "end": 412
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 404,
              "end": 412
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 417,
                "end": 422
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "activeFocusMode",
                    "loc": {
                      "start": 433,
                      "end": 448
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "mode",
                          "loc": {
                            "start": 463,
                            "end": 467
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filters",
                                "loc": {
                                  "start": 486,
                                  "end": 493
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 516,
                                        "end": 518
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 516,
                                      "end": 518
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 539,
                                        "end": 549
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 539,
                                      "end": 549
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 570,
                                        "end": 573
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 600,
                                              "end": 602
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 600,
                                            "end": 602
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 627,
                                              "end": 637
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 627,
                                            "end": 637
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 662,
                                              "end": 665
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 662,
                                            "end": 665
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 690,
                                              "end": 699
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 690,
                                            "end": 699
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 724,
                                              "end": 736
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 767,
                                                    "end": 769
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 767,
                                                  "end": 769
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 798,
                                                    "end": 806
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 798,
                                                  "end": 806
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 835,
                                                    "end": 846
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 835,
                                                  "end": 846
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 737,
                                              "end": 872
                                            }
                                          },
                                          "loc": {
                                            "start": 724,
                                            "end": 872
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 897,
                                              "end": 900
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isOwn",
                                                  "loc": {
                                                    "start": 931,
                                                    "end": 936
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 931,
                                                  "end": 936
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 965,
                                                    "end": 977
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 965,
                                                  "end": 977
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 901,
                                              "end": 1003
                                            }
                                          },
                                          "loc": {
                                            "start": 897,
                                            "end": 1003
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 574,
                                        "end": 1025
                                      }
                                    },
                                    "loc": {
                                      "start": 570,
                                      "end": 1025
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 1046,
                                        "end": 1055
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "labels",
                                            "loc": {
                                              "start": 1082,
                                              "end": 1088
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1119,
                                                    "end": 1121
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1119,
                                                  "end": 1121
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1150,
                                                    "end": 1155
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1150,
                                                  "end": 1155
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1184,
                                                    "end": 1189
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1184,
                                                  "end": 1189
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1089,
                                              "end": 1215
                                            }
                                          },
                                          "loc": {
                                            "start": 1082,
                                            "end": 1215
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderList",
                                            "loc": {
                                              "start": 1240,
                                              "end": 1252
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1283,
                                                    "end": 1285
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1283,
                                                  "end": 1285
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1314,
                                                    "end": 1324
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1314,
                                                  "end": 1324
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1353,
                                                    "end": 1363
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1353,
                                                  "end": 1363
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminders",
                                                  "loc": {
                                                    "start": 1392,
                                                    "end": 1401
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 1436,
                                                          "end": 1438
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1436,
                                                        "end": 1438
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1471,
                                                          "end": 1481
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1471,
                                                        "end": 1481
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1514,
                                                          "end": 1524
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1514,
                                                        "end": 1524
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1557,
                                                          "end": 1561
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1557,
                                                        "end": 1561
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1594,
                                                          "end": 1605
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1594,
                                                        "end": 1605
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1638,
                                                          "end": 1645
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1638,
                                                        "end": 1645
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1678,
                                                          "end": 1683
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1678,
                                                        "end": 1683
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1716,
                                                          "end": 1726
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1716,
                                                        "end": 1726
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "reminderItems",
                                                        "loc": {
                                                          "start": 1759,
                                                          "end": 1772
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "selectionSet": {
                                                        "kind": "SelectionSet",
                                                        "selections": [
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "id",
                                                              "loc": {
                                                                "start": 1811,
                                                                "end": 1813
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1811,
                                                              "end": 1813
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "created_at",
                                                              "loc": {
                                                                "start": 1850,
                                                                "end": 1860
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1850,
                                                              "end": 1860
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "updated_at",
                                                              "loc": {
                                                                "start": 1897,
                                                                "end": 1907
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1897,
                                                              "end": 1907
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 1944,
                                                                "end": 1948
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1944,
                                                              "end": 1948
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 1985,
                                                                "end": 1996
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1985,
                                                              "end": 1996
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "dueDate",
                                                              "loc": {
                                                                "start": 2033,
                                                                "end": 2040
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2033,
                                                              "end": 2040
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "index",
                                                              "loc": {
                                                                "start": 2077,
                                                                "end": 2082
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2077,
                                                              "end": 2082
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isComplete",
                                                              "loc": {
                                                                "start": 2119,
                                                                "end": 2129
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2119,
                                                              "end": 2129
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 1773,
                                                          "end": 2163
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 1759,
                                                        "end": 2163
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1402,
                                                    "end": 2193
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1392,
                                                  "end": 2193
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1253,
                                              "end": 2219
                                            }
                                          },
                                          "loc": {
                                            "start": 1240,
                                            "end": 2219
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resourceList",
                                            "loc": {
                                              "start": 2244,
                                              "end": 2256
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2287,
                                                    "end": 2289
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2287,
                                                  "end": 2289
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2318,
                                                    "end": 2328
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2318,
                                                  "end": 2328
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2357,
                                                    "end": 2369
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 2404,
                                                          "end": 2406
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2404,
                                                        "end": 2406
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2439,
                                                          "end": 2447
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2439,
                                                        "end": 2447
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2480,
                                                          "end": 2491
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2480,
                                                        "end": 2491
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2524,
                                                          "end": 2528
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2524,
                                                        "end": 2528
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2370,
                                                    "end": 2558
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2357,
                                                  "end": 2558
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "resources",
                                                  "loc": {
                                                    "start": 2587,
                                                    "end": 2596
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 2631,
                                                          "end": 2633
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2631,
                                                        "end": 2633
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 2666,
                                                          "end": 2671
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2666,
                                                        "end": 2671
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "link",
                                                        "loc": {
                                                          "start": 2704,
                                                          "end": 2708
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2704,
                                                        "end": 2708
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "usedFor",
                                                        "loc": {
                                                          "start": 2741,
                                                          "end": 2748
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2741,
                                                        "end": 2748
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "translations",
                                                        "loc": {
                                                          "start": 2781,
                                                          "end": 2793
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "selectionSet": {
                                                        "kind": "SelectionSet",
                                                        "selections": [
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "id",
                                                              "loc": {
                                                                "start": 2832,
                                                                "end": 2834
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2832,
                                                              "end": 2834
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "language",
                                                              "loc": {
                                                                "start": 2871,
                                                                "end": 2879
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2871,
                                                              "end": 2879
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 2916,
                                                                "end": 2927
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2916,
                                                              "end": 2927
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 2964,
                                                                "end": 2968
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2964,
                                                              "end": 2968
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 2794,
                                                          "end": 3002
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 2781,
                                                        "end": 3002
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2597,
                                                    "end": 3032
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2587,
                                                  "end": 3032
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2257,
                                              "end": 3058
                                            }
                                          },
                                          "loc": {
                                            "start": 2244,
                                            "end": 3058
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 3083,
                                              "end": 3091
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Schedule_common",
                                                  "loc": {
                                                    "start": 3125,
                                                    "end": 3140
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 3122,
                                                  "end": 3140
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3092,
                                              "end": 3166
                                            }
                                          },
                                          "loc": {
                                            "start": 3083,
                                            "end": 3166
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3191,
                                              "end": 3193
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3191,
                                            "end": 3193
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3218,
                                              "end": 3222
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3218,
                                            "end": 3222
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3247,
                                              "end": 3258
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3247,
                                            "end": 3258
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1056,
                                        "end": 3280
                                      }
                                    },
                                    "loc": {
                                      "start": 1046,
                                      "end": 3280
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 494,
                                  "end": 3298
                                }
                              },
                              "loc": {
                                "start": 486,
                                "end": 3298
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 3315,
                                  "end": 3321
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3344,
                                        "end": 3346
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3344,
                                      "end": 3346
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 3367,
                                        "end": 3372
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3367,
                                      "end": 3372
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 3393,
                                        "end": 3398
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3393,
                                      "end": 3398
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3322,
                                  "end": 3416
                                }
                              },
                              "loc": {
                                "start": 3315,
                                "end": 3416
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 3433,
                                  "end": 3445
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3468,
                                        "end": 3470
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3468,
                                      "end": 3470
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3491,
                                        "end": 3501
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3491,
                                      "end": 3501
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3522,
                                        "end": 3532
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3522,
                                      "end": 3532
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 3553,
                                        "end": 3562
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3589,
                                              "end": 3591
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3589,
                                            "end": 3591
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3616,
                                              "end": 3626
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3616,
                                            "end": 3626
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3651,
                                              "end": 3661
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3651,
                                            "end": 3661
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3686,
                                              "end": 3690
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3686,
                                            "end": 3690
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3715,
                                              "end": 3726
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3715,
                                            "end": 3726
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3751,
                                              "end": 3758
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3751,
                                            "end": 3758
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3783,
                                              "end": 3788
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3783,
                                            "end": 3788
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3813,
                                              "end": 3823
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3813,
                                            "end": 3823
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 3848,
                                              "end": 3861
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 3892,
                                                    "end": 3894
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3892,
                                                  "end": 3894
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3923,
                                                    "end": 3933
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3923,
                                                  "end": 3933
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3962,
                                                    "end": 3972
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3962,
                                                  "end": 3972
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 4001,
                                                    "end": 4005
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4001,
                                                  "end": 4005
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4034,
                                                    "end": 4045
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4034,
                                                  "end": 4045
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 4074,
                                                    "end": 4081
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4074,
                                                  "end": 4081
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 4110,
                                                    "end": 4115
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4110,
                                                  "end": 4115
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 4144,
                                                    "end": 4154
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4144,
                                                  "end": 4154
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3862,
                                              "end": 4180
                                            }
                                          },
                                          "loc": {
                                            "start": 3848,
                                            "end": 4180
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3563,
                                        "end": 4202
                                      }
                                    },
                                    "loc": {
                                      "start": 3553,
                                      "end": 4202
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3446,
                                  "end": 4220
                                }
                              },
                              "loc": {
                                "start": 3433,
                                "end": 4220
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 4237,
                                  "end": 4249
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4272,
                                        "end": 4274
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4272,
                                      "end": 4274
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4295,
                                        "end": 4305
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4295,
                                      "end": 4305
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4326,
                                        "end": 4338
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4365,
                                              "end": 4367
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4365,
                                            "end": 4367
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4392,
                                              "end": 4400
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4392,
                                            "end": 4400
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4425,
                                              "end": 4436
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4425,
                                            "end": 4436
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4461,
                                              "end": 4465
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4461,
                                            "end": 4465
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4339,
                                        "end": 4487
                                      }
                                    },
                                    "loc": {
                                      "start": 4326,
                                      "end": 4487
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 4508,
                                        "end": 4517
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4544,
                                              "end": 4546
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4544,
                                            "end": 4546
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4571,
                                              "end": 4576
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4571,
                                            "end": 4576
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 4601,
                                              "end": 4605
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4601,
                                            "end": 4605
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 4630,
                                              "end": 4637
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4630,
                                            "end": 4637
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 4662,
                                              "end": 4674
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 4705,
                                                    "end": 4707
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4705,
                                                  "end": 4707
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 4736,
                                                    "end": 4744
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4736,
                                                  "end": 4744
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4773,
                                                    "end": 4784
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4773,
                                                  "end": 4784
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 4813,
                                                    "end": 4817
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4813,
                                                  "end": 4817
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4675,
                                              "end": 4843
                                            }
                                          },
                                          "loc": {
                                            "start": 4662,
                                            "end": 4843
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4518,
                                        "end": 4865
                                      }
                                    },
                                    "loc": {
                                      "start": 4508,
                                      "end": 4865
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4250,
                                  "end": 4883
                                }
                              },
                              "loc": {
                                "start": 4237,
                                "end": 4883
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 4900,
                                  "end": 4908
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 4934,
                                        "end": 4949
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 4931,
                                      "end": 4949
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4909,
                                  "end": 4967
                                }
                              },
                              "loc": {
                                "start": 4900,
                                "end": 4967
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4984,
                                  "end": 4986
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4984,
                                "end": 4986
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 5003,
                                  "end": 5007
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5003,
                                "end": 5007
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5024,
                                  "end": 5035
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5024,
                                "end": 5035
                              }
                            }
                          ],
                          "loc": {
                            "start": 468,
                            "end": 5049
                          }
                        },
                        "loc": {
                          "start": 463,
                          "end": 5049
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 5062,
                            "end": 5075
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5062,
                          "end": 5075
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 5088,
                            "end": 5096
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5088,
                          "end": 5096
                        }
                      }
                    ],
                    "loc": {
                      "start": 449,
                      "end": 5106
                    }
                  },
                  "loc": {
                    "start": 433,
                    "end": 5106
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 5115,
                      "end": 5124
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5115,
                    "end": 5124
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 5133,
                      "end": 5146
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5161,
                            "end": 5163
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5161,
                          "end": 5163
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5176,
                            "end": 5186
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5176,
                          "end": 5186
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5199,
                            "end": 5209
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5199,
                          "end": 5209
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 5222,
                            "end": 5227
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5222,
                          "end": 5227
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 5240,
                            "end": 5254
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5240,
                          "end": 5254
                        }
                      }
                    ],
                    "loc": {
                      "start": 5147,
                      "end": 5264
                    }
                  },
                  "loc": {
                    "start": 5133,
                    "end": 5264
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 5273,
                      "end": 5283
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 5298,
                            "end": 5305
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5324,
                                  "end": 5326
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5324,
                                "end": 5326
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 5343,
                                  "end": 5353
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5343,
                                "end": 5353
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 5370,
                                  "end": 5373
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5396,
                                        "end": 5398
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5396,
                                      "end": 5398
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5419,
                                        "end": 5429
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5419,
                                      "end": 5429
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 5450,
                                        "end": 5453
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5450,
                                      "end": 5453
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 5474,
                                        "end": 5483
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5474,
                                      "end": 5483
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5504,
                                        "end": 5516
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5543,
                                              "end": 5545
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5543,
                                            "end": 5545
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5570,
                                              "end": 5578
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5570,
                                            "end": 5578
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5603,
                                              "end": 5614
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5603,
                                            "end": 5614
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5517,
                                        "end": 5636
                                      }
                                    },
                                    "loc": {
                                      "start": 5504,
                                      "end": 5636
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5657,
                                        "end": 5660
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 5687,
                                              "end": 5692
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5687,
                                            "end": 5692
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5717,
                                              "end": 5729
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5717,
                                            "end": 5729
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5661,
                                        "end": 5751
                                      }
                                    },
                                    "loc": {
                                      "start": 5657,
                                      "end": 5751
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5374,
                                  "end": 5769
                                }
                              },
                              "loc": {
                                "start": 5370,
                                "end": 5769
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 5786,
                                  "end": 5795
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 5818,
                                        "end": 5824
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5851,
                                              "end": 5853
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5851,
                                            "end": 5853
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 5878,
                                              "end": 5883
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5878,
                                            "end": 5883
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 5908,
                                              "end": 5913
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5908,
                                            "end": 5913
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5825,
                                        "end": 5935
                                      }
                                    },
                                    "loc": {
                                      "start": 5818,
                                      "end": 5935
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 5956,
                                        "end": 5968
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5995,
                                              "end": 5997
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5995,
                                            "end": 5997
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6022,
                                              "end": 6032
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6022,
                                            "end": 6032
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 6057,
                                              "end": 6067
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6057,
                                            "end": 6067
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 6092,
                                              "end": 6101
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 6132,
                                                    "end": 6134
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6132,
                                                  "end": 6134
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 6163,
                                                    "end": 6173
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6163,
                                                  "end": 6173
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 6202,
                                                    "end": 6212
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6202,
                                                  "end": 6212
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6241,
                                                    "end": 6245
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6241,
                                                  "end": 6245
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6274,
                                                    "end": 6285
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6274,
                                                  "end": 6285
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 6314,
                                                    "end": 6321
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6314,
                                                  "end": 6321
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 6350,
                                                    "end": 6355
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6350,
                                                  "end": 6355
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 6384,
                                                    "end": 6394
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6384,
                                                  "end": 6394
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 6423,
                                                    "end": 6436
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 6471,
                                                          "end": 6473
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6471,
                                                        "end": 6473
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 6506,
                                                          "end": 6516
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6506,
                                                        "end": 6516
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 6549,
                                                          "end": 6559
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6549,
                                                        "end": 6559
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 6592,
                                                          "end": 6596
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6592,
                                                        "end": 6596
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 6629,
                                                          "end": 6640
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6629,
                                                        "end": 6640
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 6673,
                                                          "end": 6680
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6673,
                                                        "end": 6680
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 6713,
                                                          "end": 6718
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6713,
                                                        "end": 6718
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 6751,
                                                          "end": 6761
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6751,
                                                        "end": 6761
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 6437,
                                                    "end": 6791
                                                  }
                                                },
                                                "loc": {
                                                  "start": 6423,
                                                  "end": 6791
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6102,
                                              "end": 6817
                                            }
                                          },
                                          "loc": {
                                            "start": 6092,
                                            "end": 6817
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5969,
                                        "end": 6839
                                      }
                                    },
                                    "loc": {
                                      "start": 5956,
                                      "end": 6839
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 6860,
                                        "end": 6872
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 6899,
                                              "end": 6901
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6899,
                                            "end": 6901
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6926,
                                              "end": 6936
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6926,
                                            "end": 6936
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6961,
                                              "end": 6973
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 7004,
                                                    "end": 7006
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7004,
                                                  "end": 7006
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 7035,
                                                    "end": 7043
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7035,
                                                  "end": 7043
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 7072,
                                                    "end": 7083
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7072,
                                                  "end": 7083
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 7112,
                                                    "end": 7116
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7112,
                                                  "end": 7116
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6974,
                                              "end": 7142
                                            }
                                          },
                                          "loc": {
                                            "start": 6961,
                                            "end": 7142
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 7167,
                                              "end": 7176
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 7207,
                                                    "end": 7209
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7207,
                                                  "end": 7209
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 7238,
                                                    "end": 7243
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7238,
                                                  "end": 7243
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 7272,
                                                    "end": 7276
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7272,
                                                  "end": 7276
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 7305,
                                                    "end": 7312
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7305,
                                                  "end": 7312
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 7341,
                                                    "end": 7353
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 7388,
                                                          "end": 7390
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7388,
                                                        "end": 7390
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 7423,
                                                          "end": 7431
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7423,
                                                        "end": 7431
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 7464,
                                                          "end": 7475
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7464,
                                                        "end": 7475
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 7508,
                                                          "end": 7512
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7508,
                                                        "end": 7512
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 7354,
                                                    "end": 7542
                                                  }
                                                },
                                                "loc": {
                                                  "start": 7341,
                                                  "end": 7542
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 7177,
                                              "end": 7568
                                            }
                                          },
                                          "loc": {
                                            "start": 7167,
                                            "end": 7568
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6873,
                                        "end": 7590
                                      }
                                    },
                                    "loc": {
                                      "start": 6860,
                                      "end": 7590
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 7611,
                                        "end": 7619
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 7649,
                                              "end": 7664
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 7646,
                                            "end": 7664
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7620,
                                        "end": 7686
                                      }
                                    },
                                    "loc": {
                                      "start": 7611,
                                      "end": 7686
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7707,
                                        "end": 7709
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7707,
                                      "end": 7709
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7730,
                                        "end": 7734
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7730,
                                      "end": 7734
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7755,
                                        "end": 7766
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7755,
                                      "end": 7766
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5796,
                                  "end": 7784
                                }
                              },
                              "loc": {
                                "start": 5786,
                                "end": 7784
                              }
                            }
                          ],
                          "loc": {
                            "start": 5306,
                            "end": 7798
                          }
                        },
                        "loc": {
                          "start": 5298,
                          "end": 7798
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 7811,
                            "end": 7817
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7836,
                                  "end": 7838
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7836,
                                "end": 7838
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 7855,
                                  "end": 7860
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7855,
                                "end": 7860
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 7877,
                                  "end": 7882
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7877,
                                "end": 7882
                              }
                            }
                          ],
                          "loc": {
                            "start": 7818,
                            "end": 7896
                          }
                        },
                        "loc": {
                          "start": 7811,
                          "end": 7896
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 7909,
                            "end": 7921
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7940,
                                  "end": 7942
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7940,
                                "end": 7942
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7959,
                                  "end": 7969
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7959,
                                "end": 7969
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 7986,
                                  "end": 7996
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7986,
                                "end": 7996
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 8013,
                                  "end": 8022
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 8045,
                                        "end": 8047
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8045,
                                      "end": 8047
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 8068,
                                        "end": 8078
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8068,
                                      "end": 8078
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 8099,
                                        "end": 8109
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8099,
                                      "end": 8109
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8130,
                                        "end": 8134
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8130,
                                      "end": 8134
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8155,
                                        "end": 8166
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8155,
                                      "end": 8166
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 8187,
                                        "end": 8194
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8187,
                                      "end": 8194
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8215,
                                        "end": 8220
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8215,
                                      "end": 8220
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 8241,
                                        "end": 8251
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8241,
                                      "end": 8251
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 8272,
                                        "end": 8285
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 8312,
                                              "end": 8314
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8312,
                                            "end": 8314
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 8339,
                                              "end": 8349
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8339,
                                            "end": 8349
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 8374,
                                              "end": 8384
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8374,
                                            "end": 8384
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 8409,
                                              "end": 8413
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8409,
                                            "end": 8413
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 8438,
                                              "end": 8449
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8438,
                                            "end": 8449
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 8474,
                                              "end": 8481
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8474,
                                            "end": 8481
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 8506,
                                              "end": 8511
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8506,
                                            "end": 8511
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 8536,
                                              "end": 8546
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8536,
                                            "end": 8546
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8286,
                                        "end": 8568
                                      }
                                    },
                                    "loc": {
                                      "start": 8272,
                                      "end": 8568
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8023,
                                  "end": 8586
                                }
                              },
                              "loc": {
                                "start": 8013,
                                "end": 8586
                              }
                            }
                          ],
                          "loc": {
                            "start": 7922,
                            "end": 8600
                          }
                        },
                        "loc": {
                          "start": 7909,
                          "end": 8600
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 8613,
                            "end": 8625
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 8644,
                                  "end": 8646
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8644,
                                "end": 8646
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 8663,
                                  "end": 8673
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8663,
                                "end": 8673
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 8690,
                                  "end": 8702
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 8725,
                                        "end": 8727
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8725,
                                      "end": 8727
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 8748,
                                        "end": 8756
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8748,
                                      "end": 8756
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8777,
                                        "end": 8788
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8777,
                                      "end": 8788
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8809,
                                        "end": 8813
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8809,
                                      "end": 8813
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8703,
                                  "end": 8831
                                }
                              },
                              "loc": {
                                "start": 8690,
                                "end": 8831
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 8848,
                                  "end": 8857
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 8880,
                                        "end": 8882
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8880,
                                      "end": 8882
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8903,
                                        "end": 8908
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8903,
                                      "end": 8908
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 8929,
                                        "end": 8933
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8929,
                                      "end": 8933
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 8954,
                                        "end": 8961
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8954,
                                      "end": 8961
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 8982,
                                        "end": 8994
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 9021,
                                              "end": 9023
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9021,
                                            "end": 9023
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 9048,
                                              "end": 9056
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9048,
                                            "end": 9056
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 9081,
                                              "end": 9092
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9081,
                                            "end": 9092
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 9117,
                                              "end": 9121
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9117,
                                            "end": 9121
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8995,
                                        "end": 9143
                                      }
                                    },
                                    "loc": {
                                      "start": 8982,
                                      "end": 9143
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8858,
                                  "end": 9161
                                }
                              },
                              "loc": {
                                "start": 8848,
                                "end": 9161
                              }
                            }
                          ],
                          "loc": {
                            "start": 8626,
                            "end": 9175
                          }
                        },
                        "loc": {
                          "start": 8613,
                          "end": 9175
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 9188,
                            "end": 9196
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 9218,
                                  "end": 9233
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 9215,
                                "end": 9233
                              }
                            }
                          ],
                          "loc": {
                            "start": 9197,
                            "end": 9247
                          }
                        },
                        "loc": {
                          "start": 9188,
                          "end": 9247
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 9260,
                            "end": 9262
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9260,
                          "end": 9262
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 9275,
                            "end": 9279
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9275,
                          "end": 9279
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 9292,
                            "end": 9303
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9292,
                          "end": 9303
                        }
                      }
                    ],
                    "loc": {
                      "start": 5284,
                      "end": 9313
                    }
                  },
                  "loc": {
                    "start": 5273,
                    "end": 9313
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 9322,
                      "end": 9328
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9322,
                    "end": 9328
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 9337,
                      "end": 9347
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9337,
                    "end": 9347
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 9356,
                      "end": 9358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9356,
                    "end": 9358
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 9367,
                      "end": 9376
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9367,
                    "end": 9376
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 9385,
                      "end": 9401
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9385,
                    "end": 9401
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 9410,
                      "end": 9414
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9410,
                    "end": 9414
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 9423,
                      "end": 9433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9423,
                    "end": 9433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 9442,
                      "end": 9455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9442,
                    "end": 9455
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 9464,
                      "end": 9483
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9464,
                    "end": 9483
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 9492,
                      "end": 9505
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9492,
                    "end": 9505
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 9514,
                      "end": 9533
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9514,
                    "end": 9533
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 9542,
                      "end": 9556
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9542,
                    "end": 9556
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 9565,
                      "end": 9570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9565,
                    "end": 9570
                  }
                }
              ],
              "loc": {
                "start": 423,
                "end": 9576
              }
            },
            "loc": {
              "start": 417,
              "end": 9576
            }
          }
        ],
        "loc": {
          "start": 383,
          "end": 9580
        }
      },
      "loc": {
        "start": 347,
        "end": 9580
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 40,
          "end": 42
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 40,
        "end": 42
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 43,
          "end": 53
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 43,
        "end": 53
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 54,
          "end": 64
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 54,
        "end": 64
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 65,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 65,
        "end": 74
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 75,
          "end": 82
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 75,
        "end": 82
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 83,
          "end": 91
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 83,
        "end": 91
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 92,
          "end": 102
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 109,
                "end": 111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 109,
              "end": 111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 116,
                "end": 133
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 116,
              "end": 133
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 138,
                "end": 150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 138,
              "end": 150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 155,
                "end": 165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 155,
              "end": 165
            }
          }
        ],
        "loc": {
          "start": 103,
          "end": 167
        }
      },
      "loc": {
        "start": 92,
        "end": 167
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 168,
          "end": 179
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 186,
                "end": 188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 186,
              "end": 188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 193,
                "end": 207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 212,
                "end": 220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 212,
              "end": 220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 225,
                "end": 234
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 225,
              "end": 234
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 239,
                "end": 249
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 239,
              "end": 249
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 254,
                "end": 259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 254,
              "end": 259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 264,
                "end": 271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 264,
              "end": 271
            }
          }
        ],
        "loc": {
          "start": 180,
          "end": 273
        }
      },
      "loc": {
        "start": 168,
        "end": 273
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 10,
          "end": 25
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 29,
            "end": 37
          }
        },
        "loc": {
          "start": 29,
          "end": 37
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 40,
                "end": 42
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 40,
              "end": 42
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 43,
                "end": 53
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 43,
              "end": 53
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 54,
                "end": 64
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 54,
              "end": 64
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 65,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 65,
              "end": 74
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 75,
                "end": 82
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 75,
              "end": 82
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 83,
                "end": 91
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 83,
              "end": 91
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 92,
                "end": 102
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 109,
                      "end": 111
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 109,
                    "end": 111
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 116,
                      "end": 133
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 116,
                    "end": 133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 138,
                      "end": 150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 138,
                    "end": 150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 155,
                      "end": 165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 155,
                    "end": 165
                  }
                }
              ],
              "loc": {
                "start": 103,
                "end": 167
              }
            },
            "loc": {
              "start": 92,
              "end": 167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 168,
                "end": 179
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 186,
                      "end": 188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 186,
                    "end": 188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 193,
                      "end": 207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 212,
                      "end": 220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 212,
                    "end": 220
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 225,
                      "end": 234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 225,
                    "end": 234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 239,
                      "end": 249
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 239,
                    "end": 249
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 254,
                      "end": 259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 254,
                    "end": 259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 264,
                      "end": 271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 264,
                    "end": 271
                  }
                }
              ],
              "loc": {
                "start": 180,
                "end": 273
              }
            },
            "loc": {
              "start": 168,
              "end": 273
            }
          }
        ],
        "loc": {
          "start": 38,
          "end": 275
        }
      },
      "loc": {
        "start": 1,
        "end": 275
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "switchCurrentAccount",
      "loc": {
        "start": 286,
        "end": 306
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 308,
              "end": 313
            }
          },
          "loc": {
            "start": 307,
            "end": 313
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "SwitchCurrentAccountInput",
              "loc": {
                "start": 315,
                "end": 340
              }
            },
            "loc": {
              "start": 315,
              "end": 340
            }
          },
          "loc": {
            "start": 315,
            "end": 341
          }
        },
        "directives": [],
        "loc": {
          "start": 307,
          "end": 341
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "switchCurrentAccount",
            "loc": {
              "start": 347,
              "end": 367
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 368,
                  "end": 373
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 376,
                    "end": 381
                  }
                },
                "loc": {
                  "start": 375,
                  "end": 381
                }
              },
              "loc": {
                "start": 368,
                "end": 381
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "isLoggedIn",
                  "loc": {
                    "start": 389,
                    "end": 399
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 389,
                  "end": 399
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 404,
                    "end": 412
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 404,
                  "end": 412
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 417,
                    "end": 422
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "activeFocusMode",
                        "loc": {
                          "start": 433,
                          "end": 448
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "mode",
                              "loc": {
                                "start": 463,
                                "end": 467
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filters",
                                    "loc": {
                                      "start": 486,
                                      "end": 493
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 516,
                                            "end": 518
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 516,
                                          "end": 518
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 539,
                                            "end": 549
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 539,
                                          "end": 549
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 570,
                                            "end": 573
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 600,
                                                  "end": 602
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 600,
                                                "end": 602
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 627,
                                                  "end": 637
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 627,
                                                "end": 637
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 662,
                                                  "end": 665
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 662,
                                                "end": 665
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 690,
                                                  "end": 699
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 690,
                                                "end": 699
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 724,
                                                  "end": 736
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 767,
                                                        "end": 769
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 767,
                                                      "end": 769
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 798,
                                                        "end": 806
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 798,
                                                      "end": 806
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 835,
                                                        "end": 846
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 835,
                                                      "end": 846
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 737,
                                                  "end": 872
                                                }
                                              },
                                              "loc": {
                                                "start": 724,
                                                "end": 872
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 897,
                                                  "end": 900
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isOwn",
                                                      "loc": {
                                                        "start": 931,
                                                        "end": 936
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 931,
                                                      "end": 936
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 965,
                                                        "end": 977
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 965,
                                                      "end": 977
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 901,
                                                  "end": 1003
                                                }
                                              },
                                              "loc": {
                                                "start": 897,
                                                "end": 1003
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 574,
                                            "end": 1025
                                          }
                                        },
                                        "loc": {
                                          "start": 570,
                                          "end": 1025
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 1046,
                                            "end": 1055
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "labels",
                                                "loc": {
                                                  "start": 1082,
                                                  "end": 1088
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 1119,
                                                        "end": 1121
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1119,
                                                      "end": 1121
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1150,
                                                        "end": 1155
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1150,
                                                      "end": 1155
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1184,
                                                        "end": 1189
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1184,
                                                      "end": 1189
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1089,
                                                  "end": 1215
                                                }
                                              },
                                              "loc": {
                                                "start": 1082,
                                                "end": 1215
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderList",
                                                "loc": {
                                                  "start": 1240,
                                                  "end": 1252
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 1283,
                                                        "end": 1285
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1283,
                                                      "end": 1285
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 1314,
                                                        "end": 1324
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1314,
                                                      "end": 1324
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 1353,
                                                        "end": 1363
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1353,
                                                      "end": 1363
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminders",
                                                      "loc": {
                                                        "start": 1392,
                                                        "end": 1401
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 1436,
                                                              "end": 1438
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1436,
                                                            "end": 1438
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 1471,
                                                              "end": 1481
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1471,
                                                            "end": 1481
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 1514,
                                                              "end": 1524
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1514,
                                                            "end": 1524
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 1557,
                                                              "end": 1561
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1557,
                                                            "end": 1561
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 1594,
                                                              "end": 1605
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1594,
                                                            "end": 1605
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 1638,
                                                              "end": 1645
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1638,
                                                            "end": 1645
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 1678,
                                                              "end": 1683
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1678,
                                                            "end": 1683
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 1716,
                                                              "end": 1726
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1716,
                                                            "end": 1726
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "reminderItems",
                                                            "loc": {
                                                              "start": 1759,
                                                              "end": 1772
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "selectionSet": {
                                                            "kind": "SelectionSet",
                                                            "selections": [
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "id",
                                                                  "loc": {
                                                                    "start": 1811,
                                                                    "end": 1813
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1811,
                                                                  "end": 1813
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "created_at",
                                                                  "loc": {
                                                                    "start": 1850,
                                                                    "end": 1860
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1850,
                                                                  "end": 1860
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "updated_at",
                                                                  "loc": {
                                                                    "start": 1897,
                                                                    "end": 1907
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1897,
                                                                  "end": 1907
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 1944,
                                                                    "end": 1948
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1944,
                                                                  "end": 1948
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 1985,
                                                                    "end": 1996
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1985,
                                                                  "end": 1996
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "dueDate",
                                                                  "loc": {
                                                                    "start": 2033,
                                                                    "end": 2040
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2033,
                                                                  "end": 2040
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "index",
                                                                  "loc": {
                                                                    "start": 2077,
                                                                    "end": 2082
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2077,
                                                                  "end": 2082
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "isComplete",
                                                                  "loc": {
                                                                    "start": 2119,
                                                                    "end": 2129
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2119,
                                                                  "end": 2129
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 1773,
                                                              "end": 2163
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 1759,
                                                            "end": 2163
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 1402,
                                                        "end": 2193
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 1392,
                                                      "end": 2193
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1253,
                                                  "end": 2219
                                                }
                                              },
                                              "loc": {
                                                "start": 1240,
                                                "end": 2219
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resourceList",
                                                "loc": {
                                                  "start": 2244,
                                                  "end": 2256
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 2287,
                                                        "end": 2289
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2287,
                                                      "end": 2289
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2318,
                                                        "end": 2328
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2318,
                                                      "end": 2328
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 2357,
                                                        "end": 2369
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 2404,
                                                              "end": 2406
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2404,
                                                            "end": 2406
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 2439,
                                                              "end": 2447
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2439,
                                                            "end": 2447
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 2480,
                                                              "end": 2491
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2480,
                                                            "end": 2491
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 2524,
                                                              "end": 2528
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2524,
                                                            "end": 2528
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2370,
                                                        "end": 2558
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2357,
                                                      "end": 2558
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "resources",
                                                      "loc": {
                                                        "start": 2587,
                                                        "end": 2596
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 2631,
                                                              "end": 2633
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2631,
                                                            "end": 2633
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 2666,
                                                              "end": 2671
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2666,
                                                            "end": 2671
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "link",
                                                            "loc": {
                                                              "start": 2704,
                                                              "end": 2708
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2704,
                                                            "end": 2708
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "usedFor",
                                                            "loc": {
                                                              "start": 2741,
                                                              "end": 2748
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2741,
                                                            "end": 2748
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "translations",
                                                            "loc": {
                                                              "start": 2781,
                                                              "end": 2793
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "selectionSet": {
                                                            "kind": "SelectionSet",
                                                            "selections": [
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "id",
                                                                  "loc": {
                                                                    "start": 2832,
                                                                    "end": 2834
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2832,
                                                                  "end": 2834
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "language",
                                                                  "loc": {
                                                                    "start": 2871,
                                                                    "end": 2879
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2871,
                                                                  "end": 2879
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 2916,
                                                                    "end": 2927
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2916,
                                                                  "end": 2927
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 2964,
                                                                    "end": 2968
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2964,
                                                                  "end": 2968
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 2794,
                                                              "end": 3002
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 2781,
                                                            "end": 3002
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2597,
                                                        "end": 3032
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2587,
                                                      "end": 3032
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2257,
                                                  "end": 3058
                                                }
                                              },
                                              "loc": {
                                                "start": 2244,
                                                "end": 3058
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 3083,
                                                  "end": 3091
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "FragmentSpread",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "Schedule_common",
                                                      "loc": {
                                                        "start": 3125,
                                                        "end": 3140
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3122,
                                                      "end": 3140
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3092,
                                                  "end": 3166
                                                }
                                              },
                                              "loc": {
                                                "start": 3083,
                                                "end": 3166
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3191,
                                                  "end": 3193
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3191,
                                                "end": 3193
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3218,
                                                  "end": 3222
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3218,
                                                "end": 3222
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3247,
                                                  "end": 3258
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3247,
                                                "end": 3258
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1056,
                                            "end": 3280
                                          }
                                        },
                                        "loc": {
                                          "start": 1046,
                                          "end": 3280
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 494,
                                      "end": 3298
                                    }
                                  },
                                  "loc": {
                                    "start": 486,
                                    "end": 3298
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 3315,
                                      "end": 3321
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 3344,
                                            "end": 3346
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3344,
                                          "end": 3346
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 3367,
                                            "end": 3372
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3367,
                                          "end": 3372
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 3393,
                                            "end": 3398
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3393,
                                          "end": 3398
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3322,
                                      "end": 3416
                                    }
                                  },
                                  "loc": {
                                    "start": 3315,
                                    "end": 3416
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 3433,
                                      "end": 3445
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 3468,
                                            "end": 3470
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3468,
                                          "end": 3470
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3491,
                                            "end": 3501
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3491,
                                          "end": 3501
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 3522,
                                            "end": 3532
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3522,
                                          "end": 3532
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 3553,
                                            "end": 3562
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3589,
                                                  "end": 3591
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3589,
                                                "end": 3591
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 3616,
                                                  "end": 3626
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3616,
                                                "end": 3626
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 3651,
                                                  "end": 3661
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3651,
                                                "end": 3661
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3686,
                                                  "end": 3690
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3686,
                                                "end": 3690
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3715,
                                                  "end": 3726
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3715,
                                                "end": 3726
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 3751,
                                                  "end": 3758
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3751,
                                                "end": 3758
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 3783,
                                                  "end": 3788
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3783,
                                                "end": 3788
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 3813,
                                                  "end": 3823
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3813,
                                                "end": 3823
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 3848,
                                                  "end": 3861
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 3892,
                                                        "end": 3894
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3892,
                                                      "end": 3894
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 3923,
                                                        "end": 3933
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3923,
                                                      "end": 3933
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 3962,
                                                        "end": 3972
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3962,
                                                      "end": 3972
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 4001,
                                                        "end": 4005
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4001,
                                                      "end": 4005
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4034,
                                                        "end": 4045
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4034,
                                                      "end": 4045
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 4074,
                                                        "end": 4081
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4074,
                                                      "end": 4081
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 4110,
                                                        "end": 4115
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4110,
                                                      "end": 4115
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 4144,
                                                        "end": 4154
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4144,
                                                      "end": 4154
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3862,
                                                  "end": 4180
                                                }
                                              },
                                              "loc": {
                                                "start": 3848,
                                                "end": 4180
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3563,
                                            "end": 4202
                                          }
                                        },
                                        "loc": {
                                          "start": 3553,
                                          "end": 4202
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3446,
                                      "end": 4220
                                    }
                                  },
                                  "loc": {
                                    "start": 3433,
                                    "end": 4220
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 4237,
                                      "end": 4249
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4272,
                                            "end": 4274
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4272,
                                          "end": 4274
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4295,
                                            "end": 4305
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4295,
                                          "end": 4305
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 4326,
                                            "end": 4338
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 4365,
                                                  "end": 4367
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4365,
                                                "end": 4367
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 4392,
                                                  "end": 4400
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4392,
                                                "end": 4400
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4425,
                                                  "end": 4436
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4425,
                                                "end": 4436
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4461,
                                                  "end": 4465
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4461,
                                                "end": 4465
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4339,
                                            "end": 4487
                                          }
                                        },
                                        "loc": {
                                          "start": 4326,
                                          "end": 4487
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 4508,
                                            "end": 4517
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 4544,
                                                  "end": 4546
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4544,
                                                "end": 4546
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 4571,
                                                  "end": 4576
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4571,
                                                "end": 4576
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 4601,
                                                  "end": 4605
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4601,
                                                "end": 4605
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 4630,
                                                  "end": 4637
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4630,
                                                "end": 4637
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 4662,
                                                  "end": 4674
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 4705,
                                                        "end": 4707
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4705,
                                                      "end": 4707
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 4736,
                                                        "end": 4744
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4736,
                                                      "end": 4744
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4773,
                                                        "end": 4784
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4773,
                                                      "end": 4784
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 4813,
                                                        "end": 4817
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4813,
                                                      "end": 4817
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 4675,
                                                  "end": 4843
                                                }
                                              },
                                              "loc": {
                                                "start": 4662,
                                                "end": 4843
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4518,
                                            "end": 4865
                                          }
                                        },
                                        "loc": {
                                          "start": 4508,
                                          "end": 4865
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4250,
                                      "end": 4883
                                    }
                                  },
                                  "loc": {
                                    "start": 4237,
                                    "end": 4883
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 4900,
                                      "end": 4908
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "FragmentSpread",
                                        "name": {
                                          "kind": "Name",
                                          "value": "Schedule_common",
                                          "loc": {
                                            "start": 4934,
                                            "end": 4949
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 4931,
                                          "end": 4949
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4909,
                                      "end": 4967
                                    }
                                  },
                                  "loc": {
                                    "start": 4900,
                                    "end": 4967
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4984,
                                      "end": 4986
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4984,
                                    "end": 4986
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 5003,
                                      "end": 5007
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5003,
                                    "end": 5007
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 5024,
                                      "end": 5035
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5024,
                                    "end": 5035
                                  }
                                }
                              ],
                              "loc": {
                                "start": 468,
                                "end": 5049
                              }
                            },
                            "loc": {
                              "start": 463,
                              "end": 5049
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 5062,
                                "end": 5075
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5062,
                              "end": 5075
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 5088,
                                "end": 5096
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5088,
                              "end": 5096
                            }
                          }
                        ],
                        "loc": {
                          "start": 449,
                          "end": 5106
                        }
                      },
                      "loc": {
                        "start": 433,
                        "end": 5106
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 5115,
                          "end": 5124
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5115,
                        "end": 5124
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 5133,
                          "end": 5146
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 5161,
                                "end": 5163
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5161,
                              "end": 5163
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 5176,
                                "end": 5186
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5176,
                              "end": 5186
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 5199,
                                "end": 5209
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5199,
                              "end": 5209
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 5222,
                                "end": 5227
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5222,
                              "end": 5227
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 5240,
                                "end": 5254
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5240,
                              "end": 5254
                            }
                          }
                        ],
                        "loc": {
                          "start": 5147,
                          "end": 5264
                        }
                      },
                      "loc": {
                        "start": 5133,
                        "end": 5264
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 5273,
                          "end": 5283
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "filters",
                              "loc": {
                                "start": 5298,
                                "end": 5305
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 5324,
                                      "end": 5326
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5324,
                                    "end": 5326
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 5343,
                                      "end": 5353
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5343,
                                    "end": 5353
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 5370,
                                      "end": 5373
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 5396,
                                            "end": 5398
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5396,
                                          "end": 5398
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 5419,
                                            "end": 5429
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5419,
                                          "end": 5429
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 5450,
                                            "end": 5453
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5450,
                                          "end": 5453
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 5474,
                                            "end": 5483
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5474,
                                          "end": 5483
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5504,
                                            "end": 5516
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 5543,
                                                  "end": 5545
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5543,
                                                "end": 5545
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5570,
                                                  "end": 5578
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5570,
                                                "end": 5578
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5603,
                                                  "end": 5614
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5603,
                                                "end": 5614
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5517,
                                            "end": 5636
                                          }
                                        },
                                        "loc": {
                                          "start": 5504,
                                          "end": 5636
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 5657,
                                            "end": 5660
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isOwn",
                                                "loc": {
                                                  "start": 5687,
                                                  "end": 5692
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5687,
                                                "end": 5692
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 5717,
                                                  "end": 5729
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5717,
                                                "end": 5729
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5661,
                                            "end": 5751
                                          }
                                        },
                                        "loc": {
                                          "start": 5657,
                                          "end": 5751
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5374,
                                      "end": 5769
                                    }
                                  },
                                  "loc": {
                                    "start": 5370,
                                    "end": 5769
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 5786,
                                      "end": 5795
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "labels",
                                          "loc": {
                                            "start": 5818,
                                            "end": 5824
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 5851,
                                                  "end": 5853
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5851,
                                                "end": 5853
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 5878,
                                                  "end": 5883
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5878,
                                                "end": 5883
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 5908,
                                                  "end": 5913
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5908,
                                                "end": 5913
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5825,
                                            "end": 5935
                                          }
                                        },
                                        "loc": {
                                          "start": 5818,
                                          "end": 5935
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderList",
                                          "loc": {
                                            "start": 5956,
                                            "end": 5968
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 5995,
                                                  "end": 5997
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5995,
                                                "end": 5997
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 6022,
                                                  "end": 6032
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6022,
                                                "end": 6032
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 6057,
                                                  "end": 6067
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6057,
                                                "end": 6067
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminders",
                                                "loc": {
                                                  "start": 6092,
                                                  "end": 6101
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 6132,
                                                        "end": 6134
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6132,
                                                      "end": 6134
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 6163,
                                                        "end": 6173
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6163,
                                                      "end": 6173
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 6202,
                                                        "end": 6212
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6202,
                                                      "end": 6212
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 6241,
                                                        "end": 6245
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6241,
                                                      "end": 6245
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 6274,
                                                        "end": 6285
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6274,
                                                      "end": 6285
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 6314,
                                                        "end": 6321
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6314,
                                                      "end": 6321
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 6350,
                                                        "end": 6355
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6350,
                                                      "end": 6355
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 6384,
                                                        "end": 6394
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6384,
                                                      "end": 6394
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminderItems",
                                                      "loc": {
                                                        "start": 6423,
                                                        "end": 6436
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 6471,
                                                              "end": 6473
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6471,
                                                            "end": 6473
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 6506,
                                                              "end": 6516
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6506,
                                                            "end": 6516
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 6549,
                                                              "end": 6559
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6549,
                                                            "end": 6559
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 6592,
                                                              "end": 6596
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6592,
                                                            "end": 6596
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 6629,
                                                              "end": 6640
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6629,
                                                            "end": 6640
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 6673,
                                                              "end": 6680
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6673,
                                                            "end": 6680
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 6713,
                                                              "end": 6718
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6713,
                                                            "end": 6718
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 6751,
                                                              "end": 6761
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6751,
                                                            "end": 6761
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 6437,
                                                        "end": 6791
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 6423,
                                                      "end": 6791
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6102,
                                                  "end": 6817
                                                }
                                              },
                                              "loc": {
                                                "start": 6092,
                                                "end": 6817
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5969,
                                            "end": 6839
                                          }
                                        },
                                        "loc": {
                                          "start": 5956,
                                          "end": 6839
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resourceList",
                                          "loc": {
                                            "start": 6860,
                                            "end": 6872
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 6899,
                                                  "end": 6901
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6899,
                                                "end": 6901
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 6926,
                                                  "end": 6936
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6926,
                                                "end": 6936
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 6961,
                                                  "end": 6973
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 7004,
                                                        "end": 7006
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7004,
                                                      "end": 7006
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 7035,
                                                        "end": 7043
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7035,
                                                      "end": 7043
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 7072,
                                                        "end": 7083
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7072,
                                                      "end": 7083
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 7112,
                                                        "end": 7116
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7112,
                                                      "end": 7116
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6974,
                                                  "end": 7142
                                                }
                                              },
                                              "loc": {
                                                "start": 6961,
                                                "end": 7142
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resources",
                                                "loc": {
                                                  "start": 7167,
                                                  "end": 7176
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "selectionSet": {
                                                "kind": "SelectionSet",
                                                "selections": [
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "id",
                                                      "loc": {
                                                        "start": 7207,
                                                        "end": 7209
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7207,
                                                      "end": 7209
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 7238,
                                                        "end": 7243
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7238,
                                                      "end": 7243
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "link",
                                                      "loc": {
                                                        "start": 7272,
                                                        "end": 7276
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7272,
                                                      "end": 7276
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "usedFor",
                                                      "loc": {
                                                        "start": 7305,
                                                        "end": 7312
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7305,
                                                      "end": 7312
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 7341,
                                                        "end": 7353
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "selectionSet": {
                                                      "kind": "SelectionSet",
                                                      "selections": [
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "id",
                                                            "loc": {
                                                              "start": 7388,
                                                              "end": 7390
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7388,
                                                            "end": 7390
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 7423,
                                                              "end": 7431
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7423,
                                                            "end": 7431
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 7464,
                                                              "end": 7475
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7464,
                                                            "end": 7475
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 7508,
                                                              "end": 7512
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7508,
                                                            "end": 7512
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 7354,
                                                        "end": 7542
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 7341,
                                                      "end": 7542
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 7177,
                                                  "end": 7568
                                                }
                                              },
                                              "loc": {
                                                "start": 7167,
                                                "end": 7568
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 6873,
                                            "end": 7590
                                          }
                                        },
                                        "loc": {
                                          "start": 6860,
                                          "end": 7590
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 7611,
                                            "end": 7619
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "FragmentSpread",
                                              "name": {
                                                "kind": "Name",
                                                "value": "Schedule_common",
                                                "loc": {
                                                  "start": 7649,
                                                  "end": 7664
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 7646,
                                                "end": 7664
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 7620,
                                            "end": 7686
                                          }
                                        },
                                        "loc": {
                                          "start": 7611,
                                          "end": 7686
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 7707,
                                            "end": 7709
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7707,
                                          "end": 7709
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 7730,
                                            "end": 7734
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7730,
                                          "end": 7734
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 7755,
                                            "end": 7766
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7755,
                                          "end": 7766
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5796,
                                      "end": 7784
                                    }
                                  },
                                  "loc": {
                                    "start": 5786,
                                    "end": 7784
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5306,
                                "end": 7798
                              }
                            },
                            "loc": {
                              "start": 5298,
                              "end": 7798
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 7811,
                                "end": 7817
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 7836,
                                      "end": 7838
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7836,
                                    "end": 7838
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 7855,
                                      "end": 7860
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7855,
                                    "end": 7860
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 7877,
                                      "end": 7882
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7877,
                                    "end": 7882
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7818,
                                "end": 7896
                              }
                            },
                            "loc": {
                              "start": 7811,
                              "end": 7896
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 7909,
                                "end": 7921
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 7940,
                                      "end": 7942
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7940,
                                    "end": 7942
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 7959,
                                      "end": 7969
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7959,
                                    "end": 7969
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 7986,
                                      "end": 7996
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7986,
                                    "end": 7996
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminders",
                                    "loc": {
                                      "start": 8013,
                                      "end": 8022
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 8045,
                                            "end": 8047
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8045,
                                          "end": 8047
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 8068,
                                            "end": 8078
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8068,
                                          "end": 8078
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 8099,
                                            "end": 8109
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8099,
                                          "end": 8109
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8130,
                                            "end": 8134
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8130,
                                          "end": 8134
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8155,
                                            "end": 8166
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8155,
                                          "end": 8166
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 8187,
                                            "end": 8194
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8187,
                                          "end": 8194
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8215,
                                            "end": 8220
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8215,
                                          "end": 8220
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 8241,
                                            "end": 8251
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8241,
                                          "end": 8251
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 8272,
                                            "end": 8285
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 8312,
                                                  "end": 8314
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8312,
                                                "end": 8314
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 8339,
                                                  "end": 8349
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8339,
                                                "end": 8349
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 8374,
                                                  "end": 8384
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8374,
                                                "end": 8384
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 8409,
                                                  "end": 8413
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8409,
                                                "end": 8413
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 8438,
                                                  "end": 8449
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8438,
                                                "end": 8449
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 8474,
                                                  "end": 8481
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8474,
                                                "end": 8481
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 8506,
                                                  "end": 8511
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8506,
                                                "end": 8511
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 8536,
                                                  "end": 8546
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8536,
                                                "end": 8546
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8286,
                                            "end": 8568
                                          }
                                        },
                                        "loc": {
                                          "start": 8272,
                                          "end": 8568
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8023,
                                      "end": 8586
                                    }
                                  },
                                  "loc": {
                                    "start": 8013,
                                    "end": 8586
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7922,
                                "end": 8600
                              }
                            },
                            "loc": {
                              "start": 7909,
                              "end": 8600
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 8613,
                                "end": 8625
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 8644,
                                      "end": 8646
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8644,
                                    "end": 8646
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 8663,
                                      "end": 8673
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8663,
                                    "end": 8673
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 8690,
                                      "end": 8702
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 8725,
                                            "end": 8727
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8725,
                                          "end": 8727
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 8748,
                                            "end": 8756
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8748,
                                          "end": 8756
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8777,
                                            "end": 8788
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8777,
                                          "end": 8788
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8809,
                                            "end": 8813
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8809,
                                          "end": 8813
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8703,
                                      "end": 8831
                                    }
                                  },
                                  "loc": {
                                    "start": 8690,
                                    "end": 8831
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 8848,
                                      "end": 8857
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "selectionSet": {
                                    "kind": "SelectionSet",
                                    "selections": [
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 8880,
                                            "end": 8882
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8880,
                                          "end": 8882
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8903,
                                            "end": 8908
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8903,
                                          "end": 8908
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 8929,
                                            "end": 8933
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8929,
                                          "end": 8933
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 8954,
                                            "end": 8961
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8954,
                                          "end": 8961
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 8982,
                                            "end": 8994
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "selectionSet": {
                                          "kind": "SelectionSet",
                                          "selections": [
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 9021,
                                                  "end": 9023
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9021,
                                                "end": 9023
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 9048,
                                                  "end": 9056
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9048,
                                                "end": 9056
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 9081,
                                                  "end": 9092
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9081,
                                                "end": 9092
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 9117,
                                                  "end": 9121
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9117,
                                                "end": 9121
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8995,
                                            "end": 9143
                                          }
                                        },
                                        "loc": {
                                          "start": 8982,
                                          "end": 9143
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8858,
                                      "end": 9161
                                    }
                                  },
                                  "loc": {
                                    "start": 8848,
                                    "end": 9161
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8626,
                                "end": 9175
                              }
                            },
                            "loc": {
                              "start": 8613,
                              "end": 9175
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 9188,
                                "end": 9196
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Schedule_common",
                                    "loc": {
                                      "start": 9218,
                                      "end": 9233
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 9215,
                                    "end": 9233
                                  }
                                }
                              ],
                              "loc": {
                                "start": 9197,
                                "end": 9247
                              }
                            },
                            "loc": {
                              "start": 9188,
                              "end": 9247
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 9260,
                                "end": 9262
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9260,
                              "end": 9262
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 9275,
                                "end": 9279
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9275,
                              "end": 9279
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 9292,
                                "end": 9303
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9292,
                              "end": 9303
                            }
                          }
                        ],
                        "loc": {
                          "start": 5284,
                          "end": 9313
                        }
                      },
                      "loc": {
                        "start": 5273,
                        "end": 9313
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 9322,
                          "end": 9328
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9322,
                        "end": 9328
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 9337,
                          "end": 9347
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9337,
                        "end": 9347
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 9356,
                          "end": 9358
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9356,
                        "end": 9358
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 9367,
                          "end": 9376
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9367,
                        "end": 9376
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 9385,
                          "end": 9401
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9385,
                        "end": 9401
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 9410,
                          "end": 9414
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9410,
                        "end": 9414
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 9423,
                          "end": 9433
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9423,
                        "end": 9433
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 9442,
                          "end": 9455
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9442,
                        "end": 9455
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 9464,
                          "end": 9483
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9464,
                        "end": 9483
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 9492,
                          "end": 9505
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9492,
                        "end": 9505
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 9514,
                          "end": 9533
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9514,
                        "end": 9533
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 9542,
                          "end": 9556
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9542,
                        "end": 9556
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 9565,
                          "end": 9570
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9565,
                        "end": 9570
                      }
                    }
                  ],
                  "loc": {
                    "start": 423,
                    "end": 9576
                  }
                },
                "loc": {
                  "start": 417,
                  "end": 9576
                }
              }
            ],
            "loc": {
              "start": 383,
              "end": 9580
            }
          },
          "loc": {
            "start": 347,
            "end": 9580
          }
        }
      ],
      "loc": {
        "start": 343,
        "end": 9582
      }
    },
    "loc": {
      "start": 277,
      "end": 9582
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_switchCurrentAccount"
  }
} as const;
