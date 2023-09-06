export const chatGroup_findMany = {
  "fieldName": "chatGroups",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "chatGroups",
        "loc": {
          "start": 53,
          "end": 63
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 64,
              "end": 69
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 72,
                "end": 77
              }
            },
            "loc": {
              "start": 71,
              "end": 77
            }
          },
          "loc": {
            "start": 64,
            "end": 77
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
              "value": "edges",
              "loc": {
                "start": 85,
                "end": 90
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
                    "value": "cursor",
                    "loc": {
                      "start": 101,
                      "end": 107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 101,
                    "end": 107
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 116,
                      "end": 120
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
                          "value": "chat",
                          "loc": {
                            "start": 135,
                            "end": 139
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
                                  "start": 158,
                                  "end": 160
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 158,
                                "end": 160
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 177,
                                  "end": 187
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 177,
                                "end": 187
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 204,
                                  "end": 214
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 204,
                                "end": 214
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "openToAnyoneWithInvite",
                                "loc": {
                                  "start": 231,
                                  "end": 253
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 231,
                                "end": 253
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "organization",
                                "loc": {
                                  "start": 270,
                                  "end": 282
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
                                        "start": 305,
                                        "end": 307
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 305,
                                      "end": 307
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 328,
                                        "end": 339
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 328,
                                      "end": 339
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 360,
                                        "end": 366
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 360,
                                      "end": 366
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 387,
                                        "end": 399
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 387,
                                      "end": 399
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 420,
                                        "end": 423
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
                                            "value": "canAddMembers",
                                            "loc": {
                                              "start": 450,
                                              "end": 463
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 450,
                                            "end": 463
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 488,
                                              "end": 497
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 488,
                                            "end": 497
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 522,
                                              "end": 533
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 522,
                                            "end": 533
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 558,
                                              "end": 567
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 558,
                                            "end": 567
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 592,
                                              "end": 601
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 592,
                                            "end": 601
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 626,
                                              "end": 633
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 626,
                                            "end": 633
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 658,
                                              "end": 670
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 658,
                                            "end": 670
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 695,
                                              "end": 703
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 695,
                                            "end": 703
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 728,
                                              "end": 742
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
                                                    "start": 773,
                                                    "end": 775
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 773,
                                                  "end": 775
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 804,
                                                    "end": 814
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 804,
                                                  "end": 814
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 843,
                                                    "end": 853
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 843,
                                                  "end": 853
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 882,
                                                    "end": 889
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 882,
                                                  "end": 889
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 918,
                                                    "end": 929
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 918,
                                                  "end": 929
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 743,
                                              "end": 955
                                            }
                                          },
                                          "loc": {
                                            "start": 728,
                                            "end": 955
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 424,
                                        "end": 977
                                      }
                                    },
                                    "loc": {
                                      "start": 420,
                                      "end": 977
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 283,
                                  "end": 995
                                }
                              },
                              "loc": {
                                "start": 270,
                                "end": 995
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "restrictedToRoles",
                                "loc": {
                                  "start": 1012,
                                  "end": 1029
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
                                      "value": "members",
                                      "loc": {
                                        "start": 1052,
                                        "end": 1059
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
                                              "start": 1086,
                                              "end": 1088
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1086,
                                            "end": 1088
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1113,
                                              "end": 1123
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1113,
                                            "end": 1123
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1148,
                                              "end": 1158
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1148,
                                            "end": 1158
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 1183,
                                              "end": 1190
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1183,
                                            "end": 1190
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 1215,
                                              "end": 1226
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1215,
                                            "end": 1226
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "roles",
                                            "loc": {
                                              "start": 1251,
                                              "end": 1256
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
                                                    "start": 1287,
                                                    "end": 1289
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1287,
                                                  "end": 1289
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1318,
                                                    "end": 1328
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1318,
                                                  "end": 1328
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1357,
                                                    "end": 1367
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1357,
                                                  "end": 1367
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 1396,
                                                    "end": 1400
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1396,
                                                  "end": 1400
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 1429,
                                                    "end": 1440
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1429,
                                                  "end": 1440
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "membersCount",
                                                  "loc": {
                                                    "start": 1469,
                                                    "end": 1481
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1469,
                                                  "end": 1481
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "organization",
                                                  "loc": {
                                                    "start": 1510,
                                                    "end": 1522
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
                                                          "start": 1557,
                                                          "end": 1559
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1557,
                                                        "end": 1559
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "bannerImage",
                                                        "loc": {
                                                          "start": 1592,
                                                          "end": 1603
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1592,
                                                        "end": 1603
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "handle",
                                                        "loc": {
                                                          "start": 1636,
                                                          "end": 1642
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1636,
                                                        "end": 1642
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "profileImage",
                                                        "loc": {
                                                          "start": 1675,
                                                          "end": 1687
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1675,
                                                        "end": 1687
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "you",
                                                        "loc": {
                                                          "start": 1720,
                                                          "end": 1723
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
                                                              "value": "canAddMembers",
                                                              "loc": {
                                                                "start": 1762,
                                                                "end": 1775
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1762,
                                                              "end": 1775
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "canDelete",
                                                              "loc": {
                                                                "start": 1812,
                                                                "end": 1821
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1812,
                                                              "end": 1821
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "canBookmark",
                                                              "loc": {
                                                                "start": 1858,
                                                                "end": 1869
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1858,
                                                              "end": 1869
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "canReport",
                                                              "loc": {
                                                                "start": 1906,
                                                                "end": 1915
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1906,
                                                              "end": 1915
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "canUpdate",
                                                              "loc": {
                                                                "start": 1952,
                                                                "end": 1961
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1952,
                                                              "end": 1961
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "canRead",
                                                              "loc": {
                                                                "start": 1998,
                                                                "end": 2005
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1998,
                                                              "end": 2005
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isBookmarked",
                                                              "loc": {
                                                                "start": 2042,
                                                                "end": 2054
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2042,
                                                              "end": 2054
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isViewed",
                                                              "loc": {
                                                                "start": 2091,
                                                                "end": 2099
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2091,
                                                              "end": 2099
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "yourMembership",
                                                              "loc": {
                                                                "start": 2136,
                                                                "end": 2150
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
                                                                      "start": 2193,
                                                                      "end": 2195
                                                                    }
                                                                  },
                                                                  "arguments": [],
                                                                  "directives": [],
                                                                  "loc": {
                                                                    "start": 2193,
                                                                    "end": 2195
                                                                  }
                                                                },
                                                                {
                                                                  "kind": "Field",
                                                                  "name": {
                                                                    "kind": "Name",
                                                                    "value": "created_at",
                                                                    "loc": {
                                                                      "start": 2236,
                                                                      "end": 2246
                                                                    }
                                                                  },
                                                                  "arguments": [],
                                                                  "directives": [],
                                                                  "loc": {
                                                                    "start": 2236,
                                                                    "end": 2246
                                                                  }
                                                                },
                                                                {
                                                                  "kind": "Field",
                                                                  "name": {
                                                                    "kind": "Name",
                                                                    "value": "updated_at",
                                                                    "loc": {
                                                                      "start": 2287,
                                                                      "end": 2297
                                                                    }
                                                                  },
                                                                  "arguments": [],
                                                                  "directives": [],
                                                                  "loc": {
                                                                    "start": 2287,
                                                                    "end": 2297
                                                                  }
                                                                },
                                                                {
                                                                  "kind": "Field",
                                                                  "name": {
                                                                    "kind": "Name",
                                                                    "value": "isAdmin",
                                                                    "loc": {
                                                                      "start": 2338,
                                                                      "end": 2345
                                                                    }
                                                                  },
                                                                  "arguments": [],
                                                                  "directives": [],
                                                                  "loc": {
                                                                    "start": 2338,
                                                                    "end": 2345
                                                                  }
                                                                },
                                                                {
                                                                  "kind": "Field",
                                                                  "name": {
                                                                    "kind": "Name",
                                                                    "value": "permissions",
                                                                    "loc": {
                                                                      "start": 2386,
                                                                      "end": 2397
                                                                    }
                                                                  },
                                                                  "arguments": [],
                                                                  "directives": [],
                                                                  "loc": {
                                                                    "start": 2386,
                                                                    "end": 2397
                                                                  }
                                                                }
                                                              ],
                                                              "loc": {
                                                                "start": 2151,
                                                                "end": 2435
                                                              }
                                                            },
                                                            "loc": {
                                                              "start": 2136,
                                                              "end": 2435
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 1724,
                                                          "end": 2469
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 1720,
                                                        "end": 2469
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1523,
                                                    "end": 2499
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1510,
                                                  "end": 2499
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2528,
                                                    "end": 2540
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
                                                          "start": 2575,
                                                          "end": 2577
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2575,
                                                        "end": 2577
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2610,
                                                          "end": 2618
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2610,
                                                        "end": 2618
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2651,
                                                          "end": 2662
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2651,
                                                        "end": 2662
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2541,
                                                    "end": 2692
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2528,
                                                  "end": 2692
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1257,
                                              "end": 2718
                                            }
                                          },
                                          "loc": {
                                            "start": 1251,
                                            "end": 2718
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1060,
                                        "end": 2740
                                      }
                                    },
                                    "loc": {
                                      "start": 1052,
                                      "end": 2740
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2761,
                                        "end": 2763
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2761,
                                      "end": 2763
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2784,
                                        "end": 2794
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2784,
                                      "end": 2794
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 2815,
                                        "end": 2825
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2815,
                                      "end": 2825
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2846,
                                        "end": 2850
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2846,
                                      "end": 2850
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 2871,
                                        "end": 2882
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2871,
                                      "end": 2882
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 2903,
                                        "end": 2915
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2903,
                                      "end": 2915
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "organization",
                                      "loc": {
                                        "start": 2936,
                                        "end": 2948
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
                                              "start": 2975,
                                              "end": 2977
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2975,
                                            "end": 2977
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 3002,
                                              "end": 3013
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3002,
                                            "end": 3013
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 3038,
                                              "end": 3044
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3038,
                                            "end": 3044
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 3069,
                                              "end": 3081
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3069,
                                            "end": 3081
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 3106,
                                              "end": 3109
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
                                                  "value": "canAddMembers",
                                                  "loc": {
                                                    "start": 3140,
                                                    "end": 3153
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3140,
                                                  "end": 3153
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 3182,
                                                    "end": 3191
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3182,
                                                  "end": 3191
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 3220,
                                                    "end": 3231
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3220,
                                                  "end": 3231
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 3260,
                                                    "end": 3269
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3260,
                                                  "end": 3269
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 3298,
                                                    "end": 3307
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3298,
                                                  "end": 3307
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 3336,
                                                    "end": 3343
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3336,
                                                  "end": 3343
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 3372,
                                                    "end": 3384
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3372,
                                                  "end": 3384
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 3413,
                                                    "end": 3421
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3413,
                                                  "end": 3421
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 3450,
                                                    "end": 3464
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
                                                          "start": 3499,
                                                          "end": 3501
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3499,
                                                        "end": 3501
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 3534,
                                                          "end": 3544
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3534,
                                                        "end": 3544
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 3577,
                                                          "end": 3587
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3577,
                                                        "end": 3587
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 3620,
                                                          "end": 3627
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3620,
                                                        "end": 3627
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 3660,
                                                          "end": 3671
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3660,
                                                        "end": 3671
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 3465,
                                                    "end": 3701
                                                  }
                                                },
                                                "loc": {
                                                  "start": 3450,
                                                  "end": 3701
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3110,
                                              "end": 3727
                                            }
                                          },
                                          "loc": {
                                            "start": 3106,
                                            "end": 3727
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2949,
                                        "end": 3749
                                      }
                                    },
                                    "loc": {
                                      "start": 2936,
                                      "end": 3749
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3770,
                                        "end": 3782
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
                                              "start": 3809,
                                              "end": 3811
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3809,
                                            "end": 3811
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3836,
                                              "end": 3844
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3836,
                                            "end": 3844
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3869,
                                              "end": 3880
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3869,
                                            "end": 3880
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3783,
                                        "end": 3902
                                      }
                                    },
                                    "loc": {
                                      "start": 3770,
                                      "end": 3902
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1030,
                                  "end": 3920
                                }
                              },
                              "loc": {
                                "start": 1012,
                                "end": 3920
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "participants",
                                "loc": {
                                  "start": 3937,
                                  "end": 3949
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
                                      "value": "user",
                                      "loc": {
                                        "start": 3972,
                                        "end": 3976
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
                                              "start": 4003,
                                              "end": 4005
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4003,
                                            "end": 4005
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 4030,
                                              "end": 4041
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4030,
                                            "end": 4041
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 4066,
                                              "end": 4072
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4066,
                                            "end": 4072
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBot",
                                            "loc": {
                                              "start": 4097,
                                              "end": 4102
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4097,
                                            "end": 4102
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4127,
                                              "end": 4131
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4127,
                                            "end": 4131
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 4156,
                                              "end": 4168
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4156,
                                            "end": 4168
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3977,
                                        "end": 4190
                                      }
                                    },
                                    "loc": {
                                      "start": 3972,
                                      "end": 4190
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4211,
                                        "end": 4213
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4211,
                                      "end": 4213
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4234,
                                        "end": 4244
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4234,
                                      "end": 4244
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4265,
                                        "end": 4275
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4265,
                                      "end": 4275
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3950,
                                  "end": 4293
                                }
                              },
                              "loc": {
                                "start": 3937,
                                "end": 4293
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "participantsCount",
                                "loc": {
                                  "start": 4310,
                                  "end": 4327
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4310,
                                "end": 4327
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "invitesCount",
                                "loc": {
                                  "start": 4344,
                                  "end": 4356
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4344,
                                "end": 4356
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 4373,
                                  "end": 4376
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
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 4399,
                                        "end": 4408
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4399,
                                      "end": 4408
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canInvite",
                                      "loc": {
                                        "start": 4429,
                                        "end": 4438
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4429,
                                      "end": 4438
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 4459,
                                        "end": 4468
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4459,
                                      "end": 4468
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4377,
                                  "end": 4486
                                }
                              },
                              "loc": {
                                "start": 4373,
                                "end": 4486
                              }
                            }
                          ],
                          "loc": {
                            "start": 140,
                            "end": 4500
                          }
                        },
                        "loc": {
                          "start": 135,
                          "end": 4500
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "chatsCount",
                          "loc": {
                            "start": 4513,
                            "end": 4523
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4513,
                          "end": 4523
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "participants",
                          "loc": {
                            "start": 4536,
                            "end": 4548
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
                                "value": "user",
                                "loc": {
                                  "start": 4567,
                                  "end": 4571
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
                                        "start": 4594,
                                        "end": 4596
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4594,
                                      "end": 4596
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 4617,
                                        "end": 4628
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4617,
                                      "end": 4628
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 4649,
                                        "end": 4655
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4649,
                                      "end": 4655
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBot",
                                      "loc": {
                                        "start": 4676,
                                        "end": 4681
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4676,
                                      "end": 4681
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4702,
                                        "end": 4706
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4702,
                                      "end": 4706
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 4727,
                                        "end": 4739
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4727,
                                      "end": 4739
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4572,
                                  "end": 4757
                                }
                              },
                              "loc": {
                                "start": 4567,
                                "end": 4757
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4774,
                                  "end": 4776
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4774,
                                "end": 4776
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4793,
                                  "end": 4803
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4793,
                                "end": 4803
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4820,
                                  "end": 4830
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4820,
                                "end": 4830
                              }
                            }
                          ],
                          "loc": {
                            "start": 4549,
                            "end": 4844
                          }
                        },
                        "loc": {
                          "start": 4536,
                          "end": 4844
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "participantsCount",
                          "loc": {
                            "start": 4857,
                            "end": 4874
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4857,
                          "end": 4874
                        }
                      }
                    ],
                    "loc": {
                      "start": 121,
                      "end": 4884
                    }
                  },
                  "loc": {
                    "start": 116,
                    "end": 4884
                  }
                }
              ],
              "loc": {
                "start": 91,
                "end": 4890
              }
            },
            "loc": {
              "start": 85,
              "end": 4890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 4895,
                "end": 4903
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
                    "value": "endCursor",
                    "loc": {
                      "start": 4914,
                      "end": 4923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4914,
                    "end": 4923
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 4932,
                      "end": 4943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4932,
                    "end": 4943
                  }
                }
              ],
              "loc": {
                "start": 4904,
                "end": 4949
              }
            },
            "loc": {
              "start": 4895,
              "end": 4949
            }
          }
        ],
        "loc": {
          "start": 79,
          "end": 4953
        }
      },
      "loc": {
        "start": 53,
        "end": 4953
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {},
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "chatGroups",
      "loc": {
        "start": 7,
        "end": 17
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
              "start": 19,
              "end": 24
            }
          },
          "loc": {
            "start": 18,
            "end": 24
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ChatGroupSearchInput",
              "loc": {
                "start": 26,
                "end": 46
              }
            },
            "loc": {
              "start": 26,
              "end": 46
            }
          },
          "loc": {
            "start": 26,
            "end": 47
          }
        },
        "directives": [],
        "loc": {
          "start": 18,
          "end": 47
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
            "value": "chatGroups",
            "loc": {
              "start": 53,
              "end": 63
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 64,
                  "end": 69
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 72,
                    "end": 77
                  }
                },
                "loc": {
                  "start": 71,
                  "end": 77
                }
              },
              "loc": {
                "start": 64,
                "end": 77
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
                  "value": "edges",
                  "loc": {
                    "start": 85,
                    "end": 90
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
                        "value": "cursor",
                        "loc": {
                          "start": 101,
                          "end": 107
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 101,
                        "end": 107
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 116,
                          "end": 120
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
                              "value": "chat",
                              "loc": {
                                "start": 135,
                                "end": 139
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
                                      "start": 158,
                                      "end": 160
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 158,
                                    "end": 160
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 177,
                                      "end": 187
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 177,
                                    "end": 187
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 204,
                                      "end": 214
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 204,
                                    "end": 214
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "openToAnyoneWithInvite",
                                    "loc": {
                                      "start": 231,
                                      "end": 253
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 231,
                                    "end": 253
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "organization",
                                    "loc": {
                                      "start": 270,
                                      "end": 282
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
                                            "start": 305,
                                            "end": 307
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 305,
                                          "end": 307
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bannerImage",
                                          "loc": {
                                            "start": 328,
                                            "end": 339
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 328,
                                          "end": 339
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "handle",
                                          "loc": {
                                            "start": 360,
                                            "end": 366
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 360,
                                          "end": 366
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "profileImage",
                                          "loc": {
                                            "start": 387,
                                            "end": 399
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 387,
                                          "end": 399
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 420,
                                            "end": 423
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
                                                "value": "canAddMembers",
                                                "loc": {
                                                  "start": 450,
                                                  "end": 463
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 450,
                                                "end": 463
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canDelete",
                                                "loc": {
                                                  "start": 488,
                                                  "end": 497
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 488,
                                                "end": 497
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canBookmark",
                                                "loc": {
                                                  "start": 522,
                                                  "end": 533
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 522,
                                                "end": 533
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canReport",
                                                "loc": {
                                                  "start": 558,
                                                  "end": 567
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 558,
                                                "end": 567
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canUpdate",
                                                "loc": {
                                                  "start": 592,
                                                  "end": 601
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 592,
                                                "end": 601
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "canRead",
                                                "loc": {
                                                  "start": 626,
                                                  "end": 633
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 626,
                                                "end": 633
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 658,
                                                  "end": 670
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 658,
                                                "end": 670
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isViewed",
                                                "loc": {
                                                  "start": 695,
                                                  "end": 703
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 695,
                                                "end": 703
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "yourMembership",
                                                "loc": {
                                                  "start": 728,
                                                  "end": 742
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
                                                        "start": 773,
                                                        "end": 775
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 773,
                                                      "end": 775
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 804,
                                                        "end": 814
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 804,
                                                      "end": 814
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 843,
                                                        "end": 853
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 843,
                                                      "end": 853
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isAdmin",
                                                      "loc": {
                                                        "start": 882,
                                                        "end": 889
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 882,
                                                      "end": 889
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "permissions",
                                                      "loc": {
                                                        "start": 918,
                                                        "end": 929
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 918,
                                                      "end": 929
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 743,
                                                  "end": 955
                                                }
                                              },
                                              "loc": {
                                                "start": 728,
                                                "end": 955
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 424,
                                            "end": 977
                                          }
                                        },
                                        "loc": {
                                          "start": 420,
                                          "end": 977
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 283,
                                      "end": 995
                                    }
                                  },
                                  "loc": {
                                    "start": 270,
                                    "end": 995
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "restrictedToRoles",
                                    "loc": {
                                      "start": 1012,
                                      "end": 1029
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
                                          "value": "members",
                                          "loc": {
                                            "start": 1052,
                                            "end": 1059
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
                                                  "start": 1086,
                                                  "end": 1088
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1086,
                                                "end": 1088
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 1113,
                                                  "end": 1123
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1113,
                                                "end": 1123
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 1148,
                                                  "end": 1158
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1148,
                                                "end": 1158
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isAdmin",
                                                "loc": {
                                                  "start": 1183,
                                                  "end": 1190
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1183,
                                                "end": 1190
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "permissions",
                                                "loc": {
                                                  "start": 1215,
                                                  "end": 1226
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 1215,
                                                "end": 1226
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "roles",
                                                "loc": {
                                                  "start": 1251,
                                                  "end": 1256
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
                                                        "start": 1287,
                                                        "end": 1289
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1287,
                                                      "end": 1289
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 1318,
                                                        "end": 1328
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1318,
                                                      "end": 1328
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 1357,
                                                        "end": 1367
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1357,
                                                      "end": 1367
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 1396,
                                                        "end": 1400
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1396,
                                                      "end": 1400
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "permissions",
                                                      "loc": {
                                                        "start": 1429,
                                                        "end": 1440
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1429,
                                                      "end": 1440
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "membersCount",
                                                      "loc": {
                                                        "start": 1469,
                                                        "end": 1481
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1469,
                                                      "end": 1481
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "organization",
                                                      "loc": {
                                                        "start": 1510,
                                                        "end": 1522
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
                                                              "start": 1557,
                                                              "end": 1559
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1557,
                                                            "end": 1559
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "bannerImage",
                                                            "loc": {
                                                              "start": 1592,
                                                              "end": 1603
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1592,
                                                            "end": 1603
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "handle",
                                                            "loc": {
                                                              "start": 1636,
                                                              "end": 1642
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1636,
                                                            "end": 1642
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "profileImage",
                                                            "loc": {
                                                              "start": 1675,
                                                              "end": 1687
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1675,
                                                            "end": 1687
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "you",
                                                            "loc": {
                                                              "start": 1720,
                                                              "end": 1723
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
                                                                  "value": "canAddMembers",
                                                                  "loc": {
                                                                    "start": 1762,
                                                                    "end": 1775
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1762,
                                                                  "end": 1775
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "canDelete",
                                                                  "loc": {
                                                                    "start": 1812,
                                                                    "end": 1821
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1812,
                                                                  "end": 1821
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "canBookmark",
                                                                  "loc": {
                                                                    "start": 1858,
                                                                    "end": 1869
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1858,
                                                                  "end": 1869
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "canReport",
                                                                  "loc": {
                                                                    "start": 1906,
                                                                    "end": 1915
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1906,
                                                                  "end": 1915
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "canUpdate",
                                                                  "loc": {
                                                                    "start": 1952,
                                                                    "end": 1961
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1952,
                                                                  "end": 1961
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "canRead",
                                                                  "loc": {
                                                                    "start": 1998,
                                                                    "end": 2005
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1998,
                                                                  "end": 2005
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "isBookmarked",
                                                                  "loc": {
                                                                    "start": 2042,
                                                                    "end": 2054
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2042,
                                                                  "end": 2054
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "isViewed",
                                                                  "loc": {
                                                                    "start": 2091,
                                                                    "end": 2099
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2091,
                                                                  "end": 2099
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "yourMembership",
                                                                  "loc": {
                                                                    "start": 2136,
                                                                    "end": 2150
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
                                                                          "start": 2193,
                                                                          "end": 2195
                                                                        }
                                                                      },
                                                                      "arguments": [],
                                                                      "directives": [],
                                                                      "loc": {
                                                                        "start": 2193,
                                                                        "end": 2195
                                                                      }
                                                                    },
                                                                    {
                                                                      "kind": "Field",
                                                                      "name": {
                                                                        "kind": "Name",
                                                                        "value": "created_at",
                                                                        "loc": {
                                                                          "start": 2236,
                                                                          "end": 2246
                                                                        }
                                                                      },
                                                                      "arguments": [],
                                                                      "directives": [],
                                                                      "loc": {
                                                                        "start": 2236,
                                                                        "end": 2246
                                                                      }
                                                                    },
                                                                    {
                                                                      "kind": "Field",
                                                                      "name": {
                                                                        "kind": "Name",
                                                                        "value": "updated_at",
                                                                        "loc": {
                                                                          "start": 2287,
                                                                          "end": 2297
                                                                        }
                                                                      },
                                                                      "arguments": [],
                                                                      "directives": [],
                                                                      "loc": {
                                                                        "start": 2287,
                                                                        "end": 2297
                                                                      }
                                                                    },
                                                                    {
                                                                      "kind": "Field",
                                                                      "name": {
                                                                        "kind": "Name",
                                                                        "value": "isAdmin",
                                                                        "loc": {
                                                                          "start": 2338,
                                                                          "end": 2345
                                                                        }
                                                                      },
                                                                      "arguments": [],
                                                                      "directives": [],
                                                                      "loc": {
                                                                        "start": 2338,
                                                                        "end": 2345
                                                                      }
                                                                    },
                                                                    {
                                                                      "kind": "Field",
                                                                      "name": {
                                                                        "kind": "Name",
                                                                        "value": "permissions",
                                                                        "loc": {
                                                                          "start": 2386,
                                                                          "end": 2397
                                                                        }
                                                                      },
                                                                      "arguments": [],
                                                                      "directives": [],
                                                                      "loc": {
                                                                        "start": 2386,
                                                                        "end": 2397
                                                                      }
                                                                    }
                                                                  ],
                                                                  "loc": {
                                                                    "start": 2151,
                                                                    "end": 2435
                                                                  }
                                                                },
                                                                "loc": {
                                                                  "start": 2136,
                                                                  "end": 2435
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 1724,
                                                              "end": 2469
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 1720,
                                                            "end": 2469
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 1523,
                                                        "end": 2499
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 1510,
                                                      "end": 2499
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 2528,
                                                        "end": 2540
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
                                                              "start": 2575,
                                                              "end": 2577
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2575,
                                                            "end": 2577
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 2610,
                                                              "end": 2618
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2610,
                                                            "end": 2618
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 2651,
                                                              "end": 2662
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2651,
                                                            "end": 2662
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2541,
                                                        "end": 2692
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2528,
                                                      "end": 2692
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1257,
                                                  "end": 2718
                                                }
                                              },
                                              "loc": {
                                                "start": 1251,
                                                "end": 2718
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1060,
                                            "end": 2740
                                          }
                                        },
                                        "loc": {
                                          "start": 1052,
                                          "end": 2740
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 2761,
                                            "end": 2763
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2761,
                                          "end": 2763
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 2784,
                                            "end": 2794
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2784,
                                          "end": 2794
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 2815,
                                            "end": 2825
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2815,
                                          "end": 2825
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 2846,
                                            "end": 2850
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2846,
                                          "end": 2850
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "permissions",
                                          "loc": {
                                            "start": 2871,
                                            "end": 2882
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2871,
                                          "end": 2882
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "membersCount",
                                          "loc": {
                                            "start": 2903,
                                            "end": 2915
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 2903,
                                          "end": 2915
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "organization",
                                          "loc": {
                                            "start": 2936,
                                            "end": 2948
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
                                                  "start": 2975,
                                                  "end": 2977
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 2975,
                                                "end": 2977
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bannerImage",
                                                "loc": {
                                                  "start": 3002,
                                                  "end": 3013
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3002,
                                                "end": 3013
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "handle",
                                                "loc": {
                                                  "start": 3038,
                                                  "end": 3044
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3038,
                                                "end": 3044
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "profileImage",
                                                "loc": {
                                                  "start": 3069,
                                                  "end": 3081
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3069,
                                                "end": 3081
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 3106,
                                                  "end": 3109
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
                                                      "value": "canAddMembers",
                                                      "loc": {
                                                        "start": 3140,
                                                        "end": 3153
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3140,
                                                      "end": 3153
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canDelete",
                                                      "loc": {
                                                        "start": 3182,
                                                        "end": 3191
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3182,
                                                      "end": 3191
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canBookmark",
                                                      "loc": {
                                                        "start": 3220,
                                                        "end": 3231
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3220,
                                                      "end": 3231
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canReport",
                                                      "loc": {
                                                        "start": 3260,
                                                        "end": 3269
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3260,
                                                      "end": 3269
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canUpdate",
                                                      "loc": {
                                                        "start": 3298,
                                                        "end": 3307
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3298,
                                                      "end": 3307
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "canRead",
                                                      "loc": {
                                                        "start": 3336,
                                                        "end": 3343
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3336,
                                                      "end": 3343
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 3372,
                                                        "end": 3384
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3372,
                                                      "end": 3384
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isViewed",
                                                      "loc": {
                                                        "start": 3413,
                                                        "end": 3421
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3413,
                                                      "end": 3421
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "yourMembership",
                                                      "loc": {
                                                        "start": 3450,
                                                        "end": 3464
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
                                                              "start": 3499,
                                                              "end": 3501
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 3499,
                                                            "end": 3501
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 3534,
                                                              "end": 3544
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 3534,
                                                            "end": 3544
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 3577,
                                                              "end": 3587
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 3577,
                                                            "end": 3587
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isAdmin",
                                                            "loc": {
                                                              "start": 3620,
                                                              "end": 3627
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 3620,
                                                            "end": 3627
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "permissions",
                                                            "loc": {
                                                              "start": 3660,
                                                              "end": 3671
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 3660,
                                                            "end": 3671
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 3465,
                                                        "end": 3701
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 3450,
                                                      "end": 3701
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3110,
                                                  "end": 3727
                                                }
                                              },
                                              "loc": {
                                                "start": 3106,
                                                "end": 3727
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 2949,
                                            "end": 3749
                                          }
                                        },
                                        "loc": {
                                          "start": 2936,
                                          "end": 3749
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 3770,
                                            "end": 3782
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
                                                  "start": 3809,
                                                  "end": 3811
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3809,
                                                "end": 3811
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 3836,
                                                  "end": 3844
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3836,
                                                "end": 3844
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3869,
                                                  "end": 3880
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3869,
                                                "end": 3880
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3783,
                                            "end": 3902
                                          }
                                        },
                                        "loc": {
                                          "start": 3770,
                                          "end": 3902
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 1030,
                                      "end": 3920
                                    }
                                  },
                                  "loc": {
                                    "start": 1012,
                                    "end": 3920
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "participants",
                                    "loc": {
                                      "start": 3937,
                                      "end": 3949
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
                                          "value": "user",
                                          "loc": {
                                            "start": 3972,
                                            "end": 3976
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
                                                  "start": 4003,
                                                  "end": 4005
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4003,
                                                "end": 4005
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bannerImage",
                                                "loc": {
                                                  "start": 4030,
                                                  "end": 4041
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4030,
                                                "end": 4041
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "handle",
                                                "loc": {
                                                  "start": 4066,
                                                  "end": 4072
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4066,
                                                "end": 4072
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBot",
                                                "loc": {
                                                  "start": 4097,
                                                  "end": 4102
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4097,
                                                "end": 4102
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4127,
                                                  "end": 4131
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4127,
                                                "end": 4131
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "profileImage",
                                                "loc": {
                                                  "start": 4156,
                                                  "end": 4168
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4156,
                                                "end": 4168
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3977,
                                            "end": 4190
                                          }
                                        },
                                        "loc": {
                                          "start": 3972,
                                          "end": 4190
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 4211,
                                            "end": 4213
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4211,
                                          "end": 4213
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4234,
                                            "end": 4244
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4234,
                                          "end": 4244
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 4265,
                                            "end": 4275
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4265,
                                          "end": 4275
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3950,
                                      "end": 4293
                                    }
                                  },
                                  "loc": {
                                    "start": 3937,
                                    "end": 4293
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "participantsCount",
                                    "loc": {
                                      "start": 4310,
                                      "end": 4327
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4310,
                                    "end": 4327
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "invitesCount",
                                    "loc": {
                                      "start": 4344,
                                      "end": 4356
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4344,
                                    "end": 4356
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "you",
                                    "loc": {
                                      "start": 4373,
                                      "end": 4376
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
                                          "value": "canDelete",
                                          "loc": {
                                            "start": 4399,
                                            "end": 4408
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4399,
                                          "end": 4408
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "canInvite",
                                          "loc": {
                                            "start": 4429,
                                            "end": 4438
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4429,
                                          "end": 4438
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "canUpdate",
                                          "loc": {
                                            "start": 4459,
                                            "end": 4468
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4459,
                                          "end": 4468
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4377,
                                      "end": 4486
                                    }
                                  },
                                  "loc": {
                                    "start": 4373,
                                    "end": 4486
                                  }
                                }
                              ],
                              "loc": {
                                "start": 140,
                                "end": 4500
                              }
                            },
                            "loc": {
                              "start": 135,
                              "end": 4500
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "chatsCount",
                              "loc": {
                                "start": 4513,
                                "end": 4523
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 4513,
                              "end": 4523
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "participants",
                              "loc": {
                                "start": 4536,
                                "end": 4548
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
                                    "value": "user",
                                    "loc": {
                                      "start": 4567,
                                      "end": 4571
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
                                            "start": 4594,
                                            "end": 4596
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4594,
                                          "end": 4596
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bannerImage",
                                          "loc": {
                                            "start": 4617,
                                            "end": 4628
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4617,
                                          "end": 4628
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "handle",
                                          "loc": {
                                            "start": 4649,
                                            "end": 4655
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4649,
                                          "end": 4655
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isBot",
                                          "loc": {
                                            "start": 4676,
                                            "end": 4681
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4676,
                                          "end": 4681
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 4702,
                                            "end": 4706
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4702,
                                          "end": 4706
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "profileImage",
                                          "loc": {
                                            "start": 4727,
                                            "end": 4739
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4727,
                                          "end": 4739
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4572,
                                      "end": 4757
                                    }
                                  },
                                  "loc": {
                                    "start": 4567,
                                    "end": 4757
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4774,
                                      "end": 4776
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4774,
                                    "end": 4776
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 4793,
                                      "end": 4803
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4793,
                                    "end": 4803
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 4820,
                                      "end": 4830
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4820,
                                    "end": 4830
                                  }
                                }
                              ],
                              "loc": {
                                "start": 4549,
                                "end": 4844
                              }
                            },
                            "loc": {
                              "start": 4536,
                              "end": 4844
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "participantsCount",
                              "loc": {
                                "start": 4857,
                                "end": 4874
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 4857,
                              "end": 4874
                            }
                          }
                        ],
                        "loc": {
                          "start": 121,
                          "end": 4884
                        }
                      },
                      "loc": {
                        "start": 116,
                        "end": 4884
                      }
                    }
                  ],
                  "loc": {
                    "start": 91,
                    "end": 4890
                  }
                },
                "loc": {
                  "start": 85,
                  "end": 4890
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 4895,
                    "end": 4903
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
                        "value": "endCursor",
                        "loc": {
                          "start": 4914,
                          "end": 4923
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 4914,
                        "end": 4923
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 4932,
                          "end": 4943
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 4932,
                        "end": 4943
                      }
                    }
                  ],
                  "loc": {
                    "start": 4904,
                    "end": 4949
                  }
                },
                "loc": {
                  "start": 4895,
                  "end": 4949
                }
              }
            ],
            "loc": {
              "start": 79,
              "end": 4953
            }
          },
          "loc": {
            "start": 53,
            "end": 4953
          }
        }
      ],
      "loc": {
        "start": 49,
        "end": 4955
      }
    },
    "loc": {
      "start": 1,
      "end": 4955
    }
  },
  "variableValues": {},
  "path": {
    "key": "chatGroup_findMany"
  }
} as const;
