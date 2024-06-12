export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 7710,
          "end": 7714
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7715,
              "end": 7720
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7723,
                "end": 7728
              }
            },
            "loc": {
              "start": 7722,
              "end": 7728
            }
          },
          "loc": {
            "start": 7715,
            "end": 7728
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
              "value": "recommended",
              "loc": {
                "start": 7736,
                "end": 7747
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
                    "value": "Resource_list",
                    "loc": {
                      "start": 7761,
                      "end": 7774
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7758,
                    "end": 7774
                  }
                }
              ],
              "loc": {
                "start": 7748,
                "end": 7780
              }
            },
            "loc": {
              "start": 7736,
              "end": 7780
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 7785,
                "end": 7794
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
                    "value": "Reminder_full",
                    "loc": {
                      "start": 7808,
                      "end": 7821
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7805,
                    "end": 7821
                  }
                }
              ],
              "loc": {
                "start": 7795,
                "end": 7827
              }
            },
            "loc": {
              "start": 7785,
              "end": 7827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 7832,
                "end": 7841
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
                    "value": "Resource_list",
                    "loc": {
                      "start": 7855,
                      "end": 7868
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7852,
                    "end": 7868
                  }
                }
              ],
              "loc": {
                "start": 7842,
                "end": 7874
              }
            },
            "loc": {
              "start": 7832,
              "end": 7874
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 7879,
                "end": 7888
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
                    "value": "Schedule_list",
                    "loc": {
                      "start": 7902,
                      "end": 7915
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7899,
                    "end": 7915
                  }
                }
              ],
              "loc": {
                "start": 7889,
                "end": 7921
              }
            },
            "loc": {
              "start": 7879,
              "end": 7921
            }
          }
        ],
        "loc": {
          "start": 7730,
          "end": 7925
        }
      },
      "loc": {
        "start": 7710,
        "end": 7925
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Team",
                "loc": {
                  "start": 88,
                  "end": 92
                }
              },
              "loc": {
                "start": 88,
                "end": 92
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Team_nav",
                    "loc": {
                      "start": 106,
                      "end": 114
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 103,
                    "end": 114
                  }
                }
              ],
              "loc": {
                "start": 93,
                "end": 120
              }
            },
            "loc": {
              "start": 81,
              "end": 120
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 132,
                  "end": 136
                }
              },
              "loc": {
                "start": 132,
                "end": 136
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 150,
                      "end": 158
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 147,
                    "end": 158
                  }
                }
              ],
              "loc": {
                "start": 137,
                "end": 164
              }
            },
            "loc": {
              "start": 125,
              "end": 164
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 166
        }
      },
      "loc": {
        "start": 69,
        "end": 166
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 167,
          "end": 170
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
                "start": 177,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 177,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 191,
                "end": 200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 200
            }
          }
        ],
        "loc": {
          "start": 171,
          "end": 202
        }
      },
      "loc": {
        "start": 167,
        "end": 202
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 242,
          "end": 244
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 242,
        "end": 244
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 245,
          "end": 255
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 245,
        "end": 255
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 256,
          "end": 266
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 256,
        "end": 266
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 267,
          "end": 271
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 267,
        "end": 271
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "description",
        "loc": {
          "start": 272,
          "end": 283
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 272,
        "end": 283
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "dueDate",
        "loc": {
          "start": 284,
          "end": 291
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 284,
        "end": 291
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 292,
          "end": 297
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 292,
        "end": 297
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isComplete",
        "loc": {
          "start": 298,
          "end": 308
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 298,
        "end": 308
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderItems",
        "loc": {
          "start": 309,
          "end": 322
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
                "start": 329,
                "end": 331
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 329,
              "end": 331
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 336,
                "end": 346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 336,
              "end": 346
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 351,
                "end": 361
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 351,
              "end": 361
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 366,
                "end": 370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 366,
              "end": 370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 375,
                "end": 386
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 375,
              "end": 386
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 391,
                "end": 398
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 391,
              "end": 398
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 403,
                "end": 408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 403,
              "end": 408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 413,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 413,
              "end": 423
            }
          }
        ],
        "loc": {
          "start": 323,
          "end": 425
        }
      },
      "loc": {
        "start": 309,
        "end": 425
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderList",
        "loc": {
          "start": 426,
          "end": 438
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
                "start": 445,
                "end": 447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 445,
              "end": 447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 452,
                "end": 462
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 452,
              "end": 462
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 467,
                "end": 477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 467,
              "end": 477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusMode",
              "loc": {
                "start": 482,
                "end": 491
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
                      "start": 502,
                      "end": 508
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
                            "start": 523,
                            "end": 525
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 523,
                          "end": 525
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 538,
                            "end": 543
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 538,
                          "end": 543
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 556,
                            "end": 561
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 556,
                          "end": 561
                        }
                      }
                    ],
                    "loc": {
                      "start": 509,
                      "end": 571
                    }
                  },
                  "loc": {
                    "start": 502,
                    "end": 571
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 580,
                      "end": 592
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
                            "start": 607,
                            "end": 609
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 607,
                          "end": 609
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 622,
                            "end": 632
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 622,
                          "end": 632
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 645,
                            "end": 657
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
                                  "start": 676,
                                  "end": 678
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 676,
                                "end": 678
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
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
                                "value": "description",
                                "loc": {
                                  "start": 720,
                                  "end": 731
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 720,
                                "end": 731
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 748,
                                  "end": 752
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 748,
                                "end": 752
                              }
                            }
                          ],
                          "loc": {
                            "start": 658,
                            "end": 766
                          }
                        },
                        "loc": {
                          "start": 645,
                          "end": 766
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 779,
                            "end": 788
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
                                  "start": 807,
                                  "end": 809
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 807,
                                "end": 809
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 826,
                                  "end": 831
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 826,
                                "end": 831
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 848,
                                  "end": 852
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 848,
                                "end": 852
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 869,
                                  "end": 876
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 869,
                                "end": 876
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 893,
                                  "end": 905
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
                                        "start": 928,
                                        "end": 930
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 928,
                                      "end": 930
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 951,
                                        "end": 959
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 951,
                                      "end": 959
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 980,
                                        "end": 991
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 980,
                                      "end": 991
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 1012,
                                        "end": 1016
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1012,
                                      "end": 1016
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 906,
                                  "end": 1034
                                }
                              },
                              "loc": {
                                "start": 893,
                                "end": 1034
                              }
                            }
                          ],
                          "loc": {
                            "start": 789,
                            "end": 1048
                          }
                        },
                        "loc": {
                          "start": 779,
                          "end": 1048
                        }
                      }
                    ],
                    "loc": {
                      "start": 593,
                      "end": 1058
                    }
                  },
                  "loc": {
                    "start": 580,
                    "end": 1058
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 1067,
                      "end": 1075
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
                            "start": 1093,
                            "end": 1108
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1090,
                          "end": 1108
                        }
                      }
                    ],
                    "loc": {
                      "start": 1076,
                      "end": 1118
                    }
                  },
                  "loc": {
                    "start": 1067,
                    "end": 1118
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1127,
                      "end": 1129
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1127,
                    "end": 1129
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1138,
                      "end": 1142
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1138,
                    "end": 1142
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1151,
                      "end": 1162
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1151,
                    "end": 1162
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1171,
                      "end": 1174
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
                            "start": 1189,
                            "end": 1198
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1189,
                          "end": 1198
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1211,
                            "end": 1218
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1211,
                          "end": 1218
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1231,
                            "end": 1240
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1231,
                          "end": 1240
                        }
                      }
                    ],
                    "loc": {
                      "start": 1175,
                      "end": 1250
                    }
                  },
                  "loc": {
                    "start": 1171,
                    "end": 1250
                  }
                }
              ],
              "loc": {
                "start": 492,
                "end": 1256
              }
            },
            "loc": {
              "start": 482,
              "end": 1256
            }
          }
        ],
        "loc": {
          "start": 439,
          "end": 1258
        }
      },
      "loc": {
        "start": 426,
        "end": 1258
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1298,
          "end": 1300
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1298,
        "end": 1300
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1301,
          "end": 1306
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1301,
        "end": 1306
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 1307,
          "end": 1311
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1307,
        "end": 1311
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 1312,
          "end": 1319
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1312,
        "end": 1319
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1320,
          "end": 1332
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
                "start": 1339,
                "end": 1341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1339,
              "end": 1341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1346,
                "end": 1354
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1346,
              "end": 1354
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1359,
                "end": 1370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1359,
              "end": 1370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1375,
                "end": 1379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1375,
              "end": 1379
            }
          }
        ],
        "loc": {
          "start": 1333,
          "end": 1381
        }
      },
      "loc": {
        "start": 1320,
        "end": 1381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1423,
          "end": 1425
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1423,
        "end": 1425
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1426,
          "end": 1436
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1426,
        "end": 1436
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1437,
          "end": 1447
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1437,
        "end": 1447
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 1448,
          "end": 1457
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1448,
        "end": 1457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 1458,
          "end": 1465
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1458,
        "end": 1465
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 1466,
          "end": 1474
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1466,
        "end": 1474
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 1475,
          "end": 1485
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
                "start": 1492,
                "end": 1494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1492,
              "end": 1494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 1499,
                "end": 1516
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1499,
              "end": 1516
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 1521,
                "end": 1533
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1521,
              "end": 1533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 1538,
                "end": 1548
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1538,
              "end": 1548
            }
          }
        ],
        "loc": {
          "start": 1486,
          "end": 1550
        }
      },
      "loc": {
        "start": 1475,
        "end": 1550
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 1551,
          "end": 1562
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
                "start": 1569,
                "end": 1571
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1569,
              "end": 1571
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 1576,
                "end": 1590
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1576,
              "end": 1590
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 1595,
                "end": 1603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1595,
              "end": 1603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 1608,
                "end": 1617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1608,
              "end": 1617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 1622,
                "end": 1632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1622,
              "end": 1632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 1637,
                "end": 1642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1637,
              "end": 1642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 1647,
                "end": 1654
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1647,
              "end": 1654
            }
          }
        ],
        "loc": {
          "start": 1563,
          "end": 1656
        }
      },
      "loc": {
        "start": 1551,
        "end": 1656
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1696,
          "end": 1702
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
              "value": "Label_list",
              "loc": {
                "start": 1712,
                "end": 1722
              }
            },
            "directives": [],
            "loc": {
              "start": 1709,
              "end": 1722
            }
          }
        ],
        "loc": {
          "start": 1703,
          "end": 1724
        }
      },
      "loc": {
        "start": 1696,
        "end": 1724
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModes",
        "loc": {
          "start": 1725,
          "end": 1735
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
                "start": 1742,
                "end": 1748
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
                      "start": 1759,
                      "end": 1761
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1759,
                    "end": 1761
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "color",
                    "loc": {
                      "start": 1770,
                      "end": 1775
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1770,
                    "end": 1775
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 1784,
                      "end": 1789
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1784,
                    "end": 1789
                  }
                }
              ],
              "loc": {
                "start": 1749,
                "end": 1795
              }
            },
            "loc": {
              "start": 1742,
              "end": 1795
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 1800,
                "end": 1812
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
                      "start": 1823,
                      "end": 1825
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1823,
                    "end": 1825
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1834,
                      "end": 1844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1834,
                    "end": 1844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1853,
                      "end": 1863
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1853,
                    "end": 1863
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminders",
                    "loc": {
                      "start": 1872,
                      "end": 1881
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
                            "start": 1896,
                            "end": 1898
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1896,
                          "end": 1898
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1911,
                            "end": 1921
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1911,
                          "end": 1921
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1934,
                            "end": 1944
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1934,
                          "end": 1944
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1957,
                            "end": 1961
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1957,
                          "end": 1961
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1974,
                            "end": 1985
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1974,
                          "end": 1985
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dueDate",
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
                          "value": "index",
                          "loc": {
                            "start": 2018,
                            "end": 2023
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2018,
                          "end": 2023
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 2036,
                            "end": 2046
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2036,
                          "end": 2046
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderItems",
                          "loc": {
                            "start": 2059,
                            "end": 2072
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
                                  "start": 2091,
                                  "end": 2093
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2091,
                                "end": 2093
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2110,
                                  "end": 2120
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2110,
                                "end": 2120
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2137,
                                  "end": 2147
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2137,
                                "end": 2147
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2164,
                                  "end": 2168
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2164,
                                "end": 2168
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2185,
                                  "end": 2196
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2185,
                                "end": 2196
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 2213,
                                  "end": 2220
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2213,
                                "end": 2220
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2237,
                                  "end": 2242
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2237,
                                "end": 2242
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 2259,
                                  "end": 2269
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2259,
                                "end": 2269
                              }
                            }
                          ],
                          "loc": {
                            "start": 2073,
                            "end": 2283
                          }
                        },
                        "loc": {
                          "start": 2059,
                          "end": 2283
                        }
                      }
                    ],
                    "loc": {
                      "start": 1882,
                      "end": 2293
                    }
                  },
                  "loc": {
                    "start": 1872,
                    "end": 2293
                  }
                }
              ],
              "loc": {
                "start": 1813,
                "end": 2299
              }
            },
            "loc": {
              "start": 1800,
              "end": 2299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resourceList",
              "loc": {
                "start": 2304,
                "end": 2316
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
                      "start": 2327,
                      "end": 2329
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2327,
                    "end": 2329
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2338,
                      "end": 2348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2338,
                    "end": 2348
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
                            "start": 2384,
                            "end": 2386
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2384,
                          "end": 2386
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2399,
                            "end": 2407
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2399,
                          "end": 2407
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2420,
                            "end": 2431
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2420,
                          "end": 2431
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2444,
                            "end": 2448
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2444,
                          "end": 2448
                        }
                      }
                    ],
                    "loc": {
                      "start": 2370,
                      "end": 2458
                    }
                  },
                  "loc": {
                    "start": 2357,
                    "end": 2458
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resources",
                    "loc": {
                      "start": 2467,
                      "end": 2476
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
                            "start": 2491,
                            "end": 2493
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2491,
                          "end": 2493
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 2506,
                            "end": 2511
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2506,
                          "end": 2511
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
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
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "usedFor",
                          "loc": {
                            "start": 2541,
                            "end": 2548
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2541,
                          "end": 2548
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 2561,
                            "end": 2573
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
                                  "start": 2592,
                                  "end": 2594
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2592,
                                "end": 2594
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 2611,
                                  "end": 2619
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2611,
                                "end": 2619
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2636,
                                  "end": 2647
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2636,
                                "end": 2647
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2664,
                                  "end": 2668
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2664,
                                "end": 2668
                              }
                            }
                          ],
                          "loc": {
                            "start": 2574,
                            "end": 2682
                          }
                        },
                        "loc": {
                          "start": 2561,
                          "end": 2682
                        }
                      }
                    ],
                    "loc": {
                      "start": 2477,
                      "end": 2692
                    }
                  },
                  "loc": {
                    "start": 2467,
                    "end": 2692
                  }
                }
              ],
              "loc": {
                "start": 2317,
                "end": 2698
              }
            },
            "loc": {
              "start": 2304,
              "end": 2698
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2703,
                "end": 2705
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2703,
              "end": 2705
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2710,
                "end": 2714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2710,
              "end": 2714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2719,
                "end": 2730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2719,
              "end": 2730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2735,
                "end": 2738
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
                      "start": 2749,
                      "end": 2758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2749,
                    "end": 2758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2767,
                      "end": 2774
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2767,
                    "end": 2774
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2783,
                      "end": 2792
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2783,
                    "end": 2792
                  }
                }
              ],
              "loc": {
                "start": 2739,
                "end": 2798
              }
            },
            "loc": {
              "start": 2735,
              "end": 2798
            }
          }
        ],
        "loc": {
          "start": 1736,
          "end": 2800
        }
      },
      "loc": {
        "start": 1725,
        "end": 2800
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetings",
        "loc": {
          "start": 2801,
          "end": 2809
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
                "start": 2816,
                "end": 2822
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
                    "value": "Label_list",
                    "loc": {
                      "start": 2836,
                      "end": 2846
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2833,
                    "end": 2846
                  }
                }
              ],
              "loc": {
                "start": 2823,
                "end": 2852
              }
            },
            "loc": {
              "start": 2816,
              "end": 2852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2857,
                "end": 2869
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
                      "start": 2880,
                      "end": 2882
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2880,
                    "end": 2882
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2891,
                      "end": 2899
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2891,
                    "end": 2899
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2908,
                      "end": 2919
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2908,
                    "end": 2919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "link",
                    "loc": {
                      "start": 2928,
                      "end": 2932
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2928,
                    "end": 2932
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2941,
                      "end": 2945
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2941,
                    "end": 2945
                  }
                }
              ],
              "loc": {
                "start": 2870,
                "end": 2951
              }
            },
            "loc": {
              "start": 2857,
              "end": 2951
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2956,
                "end": 2958
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2956,
              "end": 2958
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2963,
                "end": 2973
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2963,
              "end": 2973
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2978,
                "end": 2988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2978,
              "end": 2988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 2993,
                "end": 3015
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2993,
              "end": 3015
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnTeamProfile",
              "loc": {
                "start": 3020,
                "end": 3037
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3020,
              "end": 3037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "team",
              "loc": {
                "start": 3042,
                "end": 3046
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
                      "start": 3057,
                      "end": 3059
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3057,
                    "end": 3059
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 3068,
                      "end": 3079
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3068,
                    "end": 3079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 3088,
                      "end": 3094
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3088,
                    "end": 3094
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3103,
                      "end": 3115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3103,
                    "end": 3115
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 3124,
                      "end": 3127
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
                            "start": 3142,
                            "end": 3155
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3142,
                          "end": 3155
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 3168,
                            "end": 3177
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3168,
                          "end": 3177
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 3190,
                            "end": 3201
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3190,
                          "end": 3201
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 3214,
                            "end": 3223
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3214,
                          "end": 3223
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 3236,
                            "end": 3245
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3236,
                          "end": 3245
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 3258,
                            "end": 3265
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3258,
                          "end": 3265
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 3278,
                            "end": 3290
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3278,
                          "end": 3290
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 3303,
                            "end": 3311
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3303,
                          "end": 3311
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 3324,
                            "end": 3338
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
                                  "start": 3357,
                                  "end": 3359
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3357,
                                "end": 3359
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3376,
                                  "end": 3386
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3376,
                                "end": 3386
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3403,
                                  "end": 3413
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3403,
                                "end": 3413
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3430,
                                  "end": 3437
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3430,
                                "end": 3437
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3454,
                                  "end": 3465
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3454,
                                "end": 3465
                              }
                            }
                          ],
                          "loc": {
                            "start": 3339,
                            "end": 3479
                          }
                        },
                        "loc": {
                          "start": 3324,
                          "end": 3479
                        }
                      }
                    ],
                    "loc": {
                      "start": 3128,
                      "end": 3489
                    }
                  },
                  "loc": {
                    "start": 3124,
                    "end": 3489
                  }
                }
              ],
              "loc": {
                "start": 3047,
                "end": 3495
              }
            },
            "loc": {
              "start": 3042,
              "end": 3495
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 3500,
                "end": 3517
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
                      "start": 3528,
                      "end": 3535
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
                            "start": 3550,
                            "end": 3552
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3550,
                          "end": 3552
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3565,
                            "end": 3575
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3565,
                          "end": 3575
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3588,
                            "end": 3598
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3588,
                          "end": 3598
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3611,
                            "end": 3618
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3611,
                          "end": 3618
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3631,
                            "end": 3642
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3631,
                          "end": 3642
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 3655,
                            "end": 3660
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
                                  "start": 3679,
                                  "end": 3681
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3679,
                                "end": 3681
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3698,
                                  "end": 3708
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3698,
                                "end": 3708
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3725,
                                  "end": 3735
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3725,
                                "end": 3735
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3752,
                                  "end": 3756
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3752,
                                "end": 3756
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3773,
                                  "end": 3784
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3773,
                                "end": 3784
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 3801,
                                  "end": 3813
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3801,
                                "end": 3813
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "team",
                                "loc": {
                                  "start": 3830,
                                  "end": 3834
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
                                        "start": 3857,
                                        "end": 3859
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3857,
                                      "end": 3859
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 3880,
                                        "end": 3891
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3880,
                                      "end": 3891
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 3912,
                                        "end": 3918
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3912,
                                      "end": 3918
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 3939,
                                        "end": 3951
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3939,
                                      "end": 3951
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3972,
                                        "end": 3975
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
                                              "start": 4002,
                                              "end": 4015
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4002,
                                            "end": 4015
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 4040,
                                              "end": 4049
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4040,
                                            "end": 4049
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 4074,
                                              "end": 4085
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4074,
                                            "end": 4085
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 4110,
                                              "end": 4119
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4110,
                                            "end": 4119
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 4144,
                                              "end": 4153
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4144,
                                            "end": 4153
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 4178,
                                              "end": 4185
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4178,
                                            "end": 4185
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 4210,
                                              "end": 4222
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4210,
                                            "end": 4222
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 4247,
                                              "end": 4255
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4247,
                                            "end": 4255
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 4280,
                                              "end": 4294
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
                                                    "start": 4325,
                                                    "end": 4327
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4325,
                                                  "end": 4327
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 4356,
                                                    "end": 4366
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4356,
                                                  "end": 4366
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 4395,
                                                    "end": 4405
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4395,
                                                  "end": 4405
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 4434,
                                                    "end": 4441
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4434,
                                                  "end": 4441
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 4470,
                                                    "end": 4481
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4470,
                                                  "end": 4481
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4295,
                                              "end": 4507
                                            }
                                          },
                                          "loc": {
                                            "start": 4280,
                                            "end": 4507
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3976,
                                        "end": 4529
                                      }
                                    },
                                    "loc": {
                                      "start": 3972,
                                      "end": 4529
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3835,
                                  "end": 4547
                                }
                              },
                              "loc": {
                                "start": 3830,
                                "end": 4547
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4564,
                                  "end": 4576
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
                                        "start": 4599,
                                        "end": 4601
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4599,
                                      "end": 4601
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4622,
                                        "end": 4630
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4622,
                                      "end": 4630
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4651,
                                        "end": 4662
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4651,
                                      "end": 4662
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4577,
                                  "end": 4680
                                }
                              },
                              "loc": {
                                "start": 4564,
                                "end": 4680
                              }
                            }
                          ],
                          "loc": {
                            "start": 3661,
                            "end": 4694
                          }
                        },
                        "loc": {
                          "start": 3655,
                          "end": 4694
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 4707,
                            "end": 4710
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
                                  "start": 4729,
                                  "end": 4738
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4729,
                                "end": 4738
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 4755,
                                  "end": 4764
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4755,
                                "end": 4764
                              }
                            }
                          ],
                          "loc": {
                            "start": 4711,
                            "end": 4778
                          }
                        },
                        "loc": {
                          "start": 4707,
                          "end": 4778
                        }
                      }
                    ],
                    "loc": {
                      "start": 3536,
                      "end": 4788
                    }
                  },
                  "loc": {
                    "start": 3528,
                    "end": 4788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4797,
                      "end": 4799
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4797,
                    "end": 4799
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4808,
                      "end": 4818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4808,
                    "end": 4818
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4827,
                      "end": 4837
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4827,
                    "end": 4837
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4846,
                      "end": 4850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4846,
                    "end": 4850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 4859,
                      "end": 4870
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4859,
                    "end": 4870
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 4879,
                      "end": 4891
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4879,
                    "end": 4891
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "team",
                    "loc": {
                      "start": 4900,
                      "end": 4904
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
                            "start": 4919,
                            "end": 4921
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4919,
                          "end": 4921
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 4934,
                            "end": 4945
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4934,
                          "end": 4945
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 4958,
                            "end": 4964
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4958,
                          "end": 4964
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 4977,
                            "end": 4989
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4977,
                          "end": 4989
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 5002,
                            "end": 5005
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
                                  "start": 5024,
                                  "end": 5037
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5024,
                                "end": 5037
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 5054,
                                  "end": 5063
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5054,
                                "end": 5063
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 5080,
                                  "end": 5091
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5080,
                                "end": 5091
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 5108,
                                  "end": 5117
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5108,
                                "end": 5117
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 5134,
                                  "end": 5143
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5134,
                                "end": 5143
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 5160,
                                  "end": 5167
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5160,
                                "end": 5167
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 5184,
                                  "end": 5196
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5184,
                                "end": 5196
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 5213,
                                  "end": 5221
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5213,
                                "end": 5221
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 5238,
                                  "end": 5252
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
                                        "start": 5275,
                                        "end": 5277
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5275,
                                      "end": 5277
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5298,
                                        "end": 5308
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5298,
                                      "end": 5308
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 5329,
                                        "end": 5339
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5329,
                                      "end": 5339
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 5360,
                                        "end": 5367
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5360,
                                      "end": 5367
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 5388,
                                        "end": 5399
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5388,
                                      "end": 5399
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5253,
                                  "end": 5417
                                }
                              },
                              "loc": {
                                "start": 5238,
                                "end": 5417
                              }
                            }
                          ],
                          "loc": {
                            "start": 5006,
                            "end": 5431
                          }
                        },
                        "loc": {
                          "start": 5002,
                          "end": 5431
                        }
                      }
                    ],
                    "loc": {
                      "start": 4905,
                      "end": 5441
                    }
                  },
                  "loc": {
                    "start": 4900,
                    "end": 5441
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5450,
                      "end": 5462
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
                            "start": 5477,
                            "end": 5479
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5477,
                          "end": 5479
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5492,
                            "end": 5500
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5492,
                          "end": 5500
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5513,
                            "end": 5524
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5513,
                          "end": 5524
                        }
                      }
                    ],
                    "loc": {
                      "start": 5463,
                      "end": 5534
                    }
                  },
                  "loc": {
                    "start": 5450,
                    "end": 5534
                  }
                }
              ],
              "loc": {
                "start": 3518,
                "end": 5540
              }
            },
            "loc": {
              "start": 3500,
              "end": 5540
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 5545,
                "end": 5559
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5545,
              "end": 5559
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 5564,
                "end": 5576
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5564,
              "end": 5576
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5581,
                "end": 5584
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
                      "start": 5595,
                      "end": 5604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5595,
                    "end": 5604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 5613,
                      "end": 5622
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5613,
                    "end": 5622
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5631,
                      "end": 5640
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5631,
                    "end": 5640
                  }
                }
              ],
              "loc": {
                "start": 5585,
                "end": 5646
              }
            },
            "loc": {
              "start": 5581,
              "end": 5646
            }
          }
        ],
        "loc": {
          "start": 2810,
          "end": 5648
        }
      },
      "loc": {
        "start": 2801,
        "end": 5648
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 5649,
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
              "value": "projectVersion",
              "loc": {
                "start": 5667,
                "end": 5681
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
                      "start": 5692,
                      "end": 5694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5692,
                    "end": 5694
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 5703,
                      "end": 5713
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5703,
                    "end": 5713
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5722,
                      "end": 5730
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5722,
                    "end": 5730
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5739,
                      "end": 5748
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5739,
                    "end": 5748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5757,
                      "end": 5769
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5757,
                    "end": 5769
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5778,
                      "end": 5790
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5778,
                    "end": 5790
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 5799,
                      "end": 5803
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
                            "start": 5818,
                            "end": 5820
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5818,
                          "end": 5820
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5833,
                            "end": 5842
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5833,
                          "end": 5842
                        }
                      }
                    ],
                    "loc": {
                      "start": 5804,
                      "end": 5852
                    }
                  },
                  "loc": {
                    "start": 5799,
                    "end": 5852
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5861,
                      "end": 5873
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
                            "start": 5888,
                            "end": 5890
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5888,
                          "end": 5890
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5903,
                            "end": 5911
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5903,
                          "end": 5911
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5924,
                            "end": 5935
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5924,
                          "end": 5935
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5948,
                            "end": 5952
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5948,
                          "end": 5952
                        }
                      }
                    ],
                    "loc": {
                      "start": 5874,
                      "end": 5962
                    }
                  },
                  "loc": {
                    "start": 5861,
                    "end": 5962
                  }
                }
              ],
              "loc": {
                "start": 5682,
                "end": 5968
              }
            },
            "loc": {
              "start": 5667,
              "end": 5968
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5973,
                "end": 5975
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5973,
              "end": 5975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5980,
                "end": 5989
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5980,
              "end": 5989
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 5994,
                "end": 6013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5994,
              "end": 6013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6018,
                "end": 6033
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6018,
              "end": 6033
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 6038,
                "end": 6047
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6038,
              "end": 6047
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 6052,
                "end": 6063
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6052,
              "end": 6063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 6068,
                "end": 6079
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6068,
              "end": 6079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6084,
                "end": 6088
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6084,
              "end": 6088
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 6093,
                "end": 6099
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6093,
              "end": 6099
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 6104,
                "end": 6114
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6104,
              "end": 6114
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "team",
              "loc": {
                "start": 6119,
                "end": 6123
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
                    "value": "Team_nav",
                    "loc": {
                      "start": 6137,
                      "end": 6145
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6134,
                    "end": 6145
                  }
                }
              ],
              "loc": {
                "start": 6124,
                "end": 6151
              }
            },
            "loc": {
              "start": 6119,
              "end": 6151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 6156,
                "end": 6160
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
                    "value": "User_nav",
                    "loc": {
                      "start": 6174,
                      "end": 6182
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6171,
                    "end": 6182
                  }
                }
              ],
              "loc": {
                "start": 6161,
                "end": 6188
              }
            },
            "loc": {
              "start": 6156,
              "end": 6188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6193,
                "end": 6196
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
                      "start": 6207,
                      "end": 6216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6207,
                    "end": 6216
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6225,
                      "end": 6234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6225,
                    "end": 6234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6243,
                      "end": 6250
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6243,
                    "end": 6250
                  }
                }
              ],
              "loc": {
                "start": 6197,
                "end": 6256
              }
            },
            "loc": {
              "start": 6193,
              "end": 6256
            }
          }
        ],
        "loc": {
          "start": 5661,
          "end": 6258
        }
      },
      "loc": {
        "start": 5649,
        "end": 6258
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 6259,
          "end": 6270
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
              "value": "routineVersion",
              "loc": {
                "start": 6277,
                "end": 6291
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
                      "start": 6302,
                      "end": 6304
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6302,
                    "end": 6304
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 6313,
                      "end": 6323
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6313,
                    "end": 6323
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 6332,
                      "end": 6345
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6332,
                    "end": 6345
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6354,
                      "end": 6364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6354,
                    "end": 6364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 6373,
                      "end": 6382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6373,
                    "end": 6382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6391,
                      "end": 6399
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6391,
                    "end": 6399
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6408,
                      "end": 6417
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6408,
                    "end": 6417
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6426,
                      "end": 6430
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
                            "start": 6445,
                            "end": 6447
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6445,
                          "end": 6447
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 6460,
                            "end": 6470
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6460,
                          "end": 6470
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6483,
                            "end": 6492
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6483,
                          "end": 6492
                        }
                      }
                    ],
                    "loc": {
                      "start": 6431,
                      "end": 6502
                    }
                  },
                  "loc": {
                    "start": 6426,
                    "end": 6502
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6511,
                      "end": 6523
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
                            "start": 6538,
                            "end": 6540
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6538,
                          "end": 6540
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6553,
                            "end": 6561
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6553,
                          "end": 6561
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6574,
                            "end": 6585
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6574,
                          "end": 6585
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 6598,
                            "end": 6610
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6598,
                          "end": 6610
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6623,
                            "end": 6627
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6623,
                          "end": 6627
                        }
                      }
                    ],
                    "loc": {
                      "start": 6524,
                      "end": 6637
                    }
                  },
                  "loc": {
                    "start": 6511,
                    "end": 6637
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6646,
                      "end": 6658
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6646,
                    "end": 6658
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6667,
                      "end": 6679
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6667,
                    "end": 6679
                  }
                }
              ],
              "loc": {
                "start": 6292,
                "end": 6685
              }
            },
            "loc": {
              "start": 6277,
              "end": 6685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6690,
                "end": 6692
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6690,
              "end": 6692
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6697,
                "end": 6706
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6697,
              "end": 6706
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6711,
                "end": 6730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6711,
              "end": 6730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6735,
                "end": 6750
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6735,
              "end": 6750
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 6755,
                "end": 6764
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6755,
              "end": 6764
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 6769,
                "end": 6780
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6769,
              "end": 6780
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 6785,
                "end": 6796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6785,
              "end": 6796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6801,
                "end": 6805
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6801,
              "end": 6805
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 6810,
                "end": 6816
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6810,
              "end": 6816
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 6821,
                "end": 6831
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6821,
              "end": 6831
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 6836,
                "end": 6847
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6836,
              "end": 6847
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 6852,
                "end": 6871
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6852,
              "end": 6871
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "team",
              "loc": {
                "start": 6876,
                "end": 6880
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
                    "value": "Team_nav",
                    "loc": {
                      "start": 6894,
                      "end": 6902
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6891,
                    "end": 6902
                  }
                }
              ],
              "loc": {
                "start": 6881,
                "end": 6908
              }
            },
            "loc": {
              "start": 6876,
              "end": 6908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 6913,
                "end": 6917
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
                    "value": "User_nav",
                    "loc": {
                      "start": 6931,
                      "end": 6939
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6928,
                    "end": 6939
                  }
                }
              ],
              "loc": {
                "start": 6918,
                "end": 6945
              }
            },
            "loc": {
              "start": 6913,
              "end": 6945
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6950,
                "end": 6953
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
                      "start": 6964,
                      "end": 6973
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6964,
                    "end": 6973
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6982,
                      "end": 6991
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6982,
                    "end": 6991
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7000,
                      "end": 7007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7000,
                    "end": 7007
                  }
                }
              ],
              "loc": {
                "start": 6954,
                "end": 7013
              }
            },
            "loc": {
              "start": 6950,
              "end": 7013
            }
          }
        ],
        "loc": {
          "start": 6271,
          "end": 7015
        }
      },
      "loc": {
        "start": 6259,
        "end": 7015
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7016,
          "end": 7018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7016,
        "end": 7018
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7019,
          "end": 7029
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7019,
        "end": 7029
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7030,
          "end": 7040
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7030,
        "end": 7040
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 7041,
          "end": 7050
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7041,
        "end": 7050
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 7051,
          "end": 7058
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7051,
        "end": 7058
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 7059,
          "end": 7067
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7059,
        "end": 7067
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 7068,
          "end": 7078
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
                "start": 7085,
                "end": 7087
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7085,
              "end": 7087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 7092,
                "end": 7109
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7092,
              "end": 7109
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 7114,
                "end": 7126
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7114,
              "end": 7126
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 7131,
                "end": 7141
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7131,
              "end": 7141
            }
          }
        ],
        "loc": {
          "start": 7079,
          "end": 7143
        }
      },
      "loc": {
        "start": 7068,
        "end": 7143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 7144,
          "end": 7155
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
                "start": 7162,
                "end": 7164
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7162,
              "end": 7164
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 7169,
                "end": 7183
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7169,
              "end": 7183
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 7188,
                "end": 7196
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7188,
              "end": 7196
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 7201,
                "end": 7210
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7201,
              "end": 7210
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 7215,
                "end": 7225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7215,
              "end": 7225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 7230,
                "end": 7235
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7230,
              "end": 7235
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 7240,
                "end": 7247
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7240,
              "end": 7247
            }
          }
        ],
        "loc": {
          "start": 7156,
          "end": 7249
        }
      },
      "loc": {
        "start": 7144,
        "end": 7249
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7280,
          "end": 7282
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7280,
        "end": 7282
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7283,
          "end": 7294
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7283,
        "end": 7294
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7295,
          "end": 7301
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7295,
        "end": 7301
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7302,
          "end": 7314
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7302,
        "end": 7314
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7315,
          "end": 7318
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
                "start": 7325,
                "end": 7338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7325,
              "end": 7338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7343,
                "end": 7352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7343,
              "end": 7352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7357,
                "end": 7368
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7357,
              "end": 7368
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7373,
                "end": 7382
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7373,
              "end": 7382
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7387,
                "end": 7396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7387,
              "end": 7396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7401,
                "end": 7408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7401,
              "end": 7408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7413,
                "end": 7425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7413,
              "end": 7425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7430,
                "end": 7438
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7430,
              "end": 7438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 7443,
                "end": 7457
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
                      "start": 7468,
                      "end": 7470
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7468,
                    "end": 7470
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7479,
                      "end": 7489
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7479,
                    "end": 7489
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7498,
                      "end": 7508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7498,
                    "end": 7508
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 7517,
                      "end": 7524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7517,
                    "end": 7524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 7533,
                      "end": 7544
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7533,
                    "end": 7544
                  }
                }
              ],
              "loc": {
                "start": 7458,
                "end": 7550
              }
            },
            "loc": {
              "start": 7443,
              "end": 7550
            }
          }
        ],
        "loc": {
          "start": 7319,
          "end": 7552
        }
      },
      "loc": {
        "start": 7315,
        "end": 7552
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7583,
          "end": 7585
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7583,
        "end": 7585
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7586,
          "end": 7596
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7586,
        "end": 7596
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7597,
          "end": 7607
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7597,
        "end": 7607
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7608,
          "end": 7619
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7608,
        "end": 7619
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7620,
          "end": 7626
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7620,
        "end": 7626
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7627,
          "end": 7632
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7627,
        "end": 7632
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 7633,
          "end": 7653
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7633,
        "end": 7653
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7654,
          "end": 7658
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7654,
        "end": 7658
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7659,
          "end": 7671
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7659,
        "end": 7671
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
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
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Team",
                      "loc": {
                        "start": 88,
                        "end": 92
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 92
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Team_nav",
                          "loc": {
                            "start": 106,
                            "end": 114
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 103,
                          "end": 114
                        }
                      }
                    ],
                    "loc": {
                      "start": 93,
                      "end": 120
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 120
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 132,
                        "end": 136
                      }
                    },
                    "loc": {
                      "start": 132,
                      "end": 136
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 150,
                            "end": 158
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 147,
                          "end": 158
                        }
                      }
                    ],
                    "loc": {
                      "start": 137,
                      "end": 164
                    }
                  },
                  "loc": {
                    "start": 125,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 166
              }
            },
            "loc": {
              "start": 69,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 167,
                "end": 170
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
                      "start": 177,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 177,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 191,
                      "end": 200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 200
                  }
                }
              ],
              "loc": {
                "start": 171,
                "end": 202
              }
            },
            "loc": {
              "start": 167,
              "end": 202
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 204
        }
      },
      "loc": {
        "start": 1,
        "end": 204
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 214,
          "end": 227
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 231,
            "end": 239
          }
        },
        "loc": {
          "start": 231,
          "end": 239
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
                "start": 242,
                "end": 244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 242,
              "end": 244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 245,
                "end": 255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 245,
              "end": 255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 256,
                "end": 266
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 256,
              "end": 266
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 267,
                "end": 271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 267,
              "end": 271
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 272,
                "end": 283
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 272,
              "end": 283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 284,
                "end": 291
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 284,
              "end": 291
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 292,
                "end": 297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 292,
              "end": 297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 298,
                "end": 308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 298,
              "end": 308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 309,
                "end": 322
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
                      "start": 329,
                      "end": 331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 329,
                    "end": 331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 336,
                      "end": 346
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 336,
                    "end": 346
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 351,
                      "end": 361
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 351,
                    "end": 361
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 366,
                      "end": 370
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 366,
                    "end": 370
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 375,
                      "end": 386
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 375,
                    "end": 386
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 391,
                      "end": 398
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 398
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 403,
                      "end": 408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 403,
                    "end": 408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 413,
                      "end": 423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 413,
                    "end": 423
                  }
                }
              ],
              "loc": {
                "start": 323,
                "end": 425
              }
            },
            "loc": {
              "start": 309,
              "end": 425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 426,
                "end": 438
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
                      "start": 445,
                      "end": 447
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 445,
                    "end": 447
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 452,
                      "end": 462
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 452,
                    "end": 462
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 467,
                      "end": 477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 467,
                    "end": 477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 482,
                      "end": 491
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
                            "start": 502,
                            "end": 508
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
                                  "start": 523,
                                  "end": 525
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 523,
                                "end": 525
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 538,
                                  "end": 543
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 538,
                                "end": 543
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 556,
                                  "end": 561
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 556,
                                "end": 561
                              }
                            }
                          ],
                          "loc": {
                            "start": 509,
                            "end": 571
                          }
                        },
                        "loc": {
                          "start": 502,
                          "end": 571
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 580,
                            "end": 592
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
                                  "start": 607,
                                  "end": 609
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 607,
                                "end": 609
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 622,
                                  "end": 632
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 622,
                                "end": 632
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 645,
                                  "end": 657
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
                                        "start": 676,
                                        "end": 678
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 676,
                                      "end": 678
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
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
                                      "value": "description",
                                      "loc": {
                                        "start": 720,
                                        "end": 731
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 720,
                                      "end": 731
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 748,
                                        "end": 752
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 748,
                                      "end": 752
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 658,
                                  "end": 766
                                }
                              },
                              "loc": {
                                "start": 645,
                                "end": 766
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 779,
                                  "end": 788
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
                                        "start": 807,
                                        "end": 809
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 807,
                                      "end": 809
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 826,
                                        "end": 831
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 826,
                                      "end": 831
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 848,
                                        "end": 852
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 848,
                                      "end": 852
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 869,
                                        "end": 876
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 869,
                                      "end": 876
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 893,
                                        "end": 905
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
                                              "start": 928,
                                              "end": 930
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 928,
                                            "end": 930
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 951,
                                              "end": 959
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 951,
                                            "end": 959
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 980,
                                              "end": 991
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 980,
                                            "end": 991
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1012,
                                              "end": 1016
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1012,
                                            "end": 1016
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 906,
                                        "end": 1034
                                      }
                                    },
                                    "loc": {
                                      "start": 893,
                                      "end": 1034
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 789,
                                  "end": 1048
                                }
                              },
                              "loc": {
                                "start": 779,
                                "end": 1048
                              }
                            }
                          ],
                          "loc": {
                            "start": 593,
                            "end": 1058
                          }
                        },
                        "loc": {
                          "start": 580,
                          "end": 1058
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 1067,
                            "end": 1075
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
                                  "start": 1093,
                                  "end": 1108
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1090,
                                "end": 1108
                              }
                            }
                          ],
                          "loc": {
                            "start": 1076,
                            "end": 1118
                          }
                        },
                        "loc": {
                          "start": 1067,
                          "end": 1118
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1127,
                            "end": 1129
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1127,
                          "end": 1129
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1138,
                            "end": 1142
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1138,
                          "end": 1142
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1151,
                            "end": 1162
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1151,
                          "end": 1162
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 1171,
                            "end": 1174
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
                                  "start": 1189,
                                  "end": 1198
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1189,
                                "end": 1198
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 1211,
                                  "end": 1218
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1211,
                                "end": 1218
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 1231,
                                  "end": 1240
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1231,
                                "end": 1240
                              }
                            }
                          ],
                          "loc": {
                            "start": 1175,
                            "end": 1250
                          }
                        },
                        "loc": {
                          "start": 1171,
                          "end": 1250
                        }
                      }
                    ],
                    "loc": {
                      "start": 492,
                      "end": 1256
                    }
                  },
                  "loc": {
                    "start": 482,
                    "end": 1256
                  }
                }
              ],
              "loc": {
                "start": 439,
                "end": 1258
              }
            },
            "loc": {
              "start": 426,
              "end": 1258
            }
          }
        ],
        "loc": {
          "start": 240,
          "end": 1260
        }
      },
      "loc": {
        "start": 205,
        "end": 1260
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1270,
          "end": 1283
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1287,
            "end": 1295
          }
        },
        "loc": {
          "start": 1287,
          "end": 1295
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
                "start": 1298,
                "end": 1300
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1298,
              "end": 1300
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1301,
                "end": 1306
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1301,
              "end": 1306
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1307,
                "end": 1311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1307,
              "end": 1311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1312,
                "end": 1319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1312,
              "end": 1319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1320,
                "end": 1332
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
                      "start": 1339,
                      "end": 1341
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1339,
                    "end": 1341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1346,
                      "end": 1354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1346,
                    "end": 1354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1359,
                      "end": 1370
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1359,
                    "end": 1370
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1375,
                      "end": 1379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1375,
                    "end": 1379
                  }
                }
              ],
              "loc": {
                "start": 1333,
                "end": 1381
              }
            },
            "loc": {
              "start": 1320,
              "end": 1381
            }
          }
        ],
        "loc": {
          "start": 1296,
          "end": 1383
        }
      },
      "loc": {
        "start": 1261,
        "end": 1383
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 1393,
          "end": 1408
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1412,
            "end": 1420
          }
        },
        "loc": {
          "start": 1412,
          "end": 1420
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
                "start": 1423,
                "end": 1425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1423,
              "end": 1425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1426,
                "end": 1436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1426,
              "end": 1436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1437,
                "end": 1447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1437,
              "end": 1447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1448,
                "end": 1457
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1448,
              "end": 1457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 1458,
                "end": 1465
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1458,
              "end": 1465
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 1466,
                "end": 1474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1466,
              "end": 1474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 1475,
                "end": 1485
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
                      "start": 1492,
                      "end": 1494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1492,
                    "end": 1494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 1499,
                      "end": 1516
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1499,
                    "end": 1516
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 1521,
                      "end": 1533
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1521,
                    "end": 1533
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 1538,
                      "end": 1548
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1538,
                    "end": 1548
                  }
                }
              ],
              "loc": {
                "start": 1486,
                "end": 1550
              }
            },
            "loc": {
              "start": 1475,
              "end": 1550
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 1551,
                "end": 1562
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
                      "start": 1569,
                      "end": 1571
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1569,
                    "end": 1571
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 1576,
                      "end": 1590
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1576,
                    "end": 1590
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 1595,
                      "end": 1603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1595,
                    "end": 1603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 1608,
                      "end": 1617
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1608,
                    "end": 1617
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 1622,
                      "end": 1632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1622,
                    "end": 1632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 1637,
                      "end": 1642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1637,
                    "end": 1642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 1647,
                      "end": 1654
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1647,
                    "end": 1654
                  }
                }
              ],
              "loc": {
                "start": 1563,
                "end": 1656
              }
            },
            "loc": {
              "start": 1551,
              "end": 1656
            }
          }
        ],
        "loc": {
          "start": 1421,
          "end": 1658
        }
      },
      "loc": {
        "start": 1384,
        "end": 1658
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 1668,
          "end": 1681
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1685,
            "end": 1693
          }
        },
        "loc": {
          "start": 1685,
          "end": 1693
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
              "value": "labels",
              "loc": {
                "start": 1696,
                "end": 1702
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
                    "value": "Label_list",
                    "loc": {
                      "start": 1712,
                      "end": 1722
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1709,
                    "end": 1722
                  }
                }
              ],
              "loc": {
                "start": 1703,
                "end": 1724
              }
            },
            "loc": {
              "start": 1696,
              "end": 1724
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 1725,
                "end": 1735
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
                      "start": 1742,
                      "end": 1748
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
                            "start": 1759,
                            "end": 1761
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1759,
                          "end": 1761
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 1770,
                            "end": 1775
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1770,
                          "end": 1775
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 1784,
                            "end": 1789
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1784,
                          "end": 1789
                        }
                      }
                    ],
                    "loc": {
                      "start": 1749,
                      "end": 1795
                    }
                  },
                  "loc": {
                    "start": 1742,
                    "end": 1795
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminderList",
                    "loc": {
                      "start": 1800,
                      "end": 1812
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
                            "start": 1823,
                            "end": 1825
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1823,
                          "end": 1825
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1834,
                            "end": 1844
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1834,
                          "end": 1844
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1853,
                            "end": 1863
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1853,
                          "end": 1863
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminders",
                          "loc": {
                            "start": 1872,
                            "end": 1881
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
                                  "start": 1896,
                                  "end": 1898
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1896,
                                "end": 1898
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 1911,
                                  "end": 1921
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1911,
                                "end": 1921
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 1934,
                                  "end": 1944
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1934,
                                "end": 1944
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 1957,
                                  "end": 1961
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1957,
                                "end": 1961
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 1974,
                                  "end": 1985
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1974,
                                "end": 1985
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
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
                                "value": "index",
                                "loc": {
                                  "start": 2018,
                                  "end": 2023
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2018,
                                "end": 2023
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 2036,
                                  "end": 2046
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2036,
                                "end": 2046
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderItems",
                                "loc": {
                                  "start": 2059,
                                  "end": 2072
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
                                        "start": 2091,
                                        "end": 2093
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2091,
                                      "end": 2093
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2110,
                                        "end": 2120
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2110,
                                      "end": 2120
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 2137,
                                        "end": 2147
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2137,
                                      "end": 2147
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2164,
                                        "end": 2168
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2164,
                                      "end": 2168
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2185,
                                        "end": 2196
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2185,
                                      "end": 2196
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 2213,
                                        "end": 2220
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2213,
                                      "end": 2220
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 2237,
                                        "end": 2242
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2237,
                                      "end": 2242
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 2259,
                                        "end": 2269
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2259,
                                      "end": 2269
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2073,
                                  "end": 2283
                                }
                              },
                              "loc": {
                                "start": 2059,
                                "end": 2283
                              }
                            }
                          ],
                          "loc": {
                            "start": 1882,
                            "end": 2293
                          }
                        },
                        "loc": {
                          "start": 1872,
                          "end": 2293
                        }
                      }
                    ],
                    "loc": {
                      "start": 1813,
                      "end": 2299
                    }
                  },
                  "loc": {
                    "start": 1800,
                    "end": 2299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 2304,
                      "end": 2316
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
                            "start": 2327,
                            "end": 2329
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2327,
                          "end": 2329
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2338,
                            "end": 2348
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2338,
                          "end": 2348
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
                                  "start": 2384,
                                  "end": 2386
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2384,
                                "end": 2386
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 2399,
                                  "end": 2407
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2399,
                                "end": 2407
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2420,
                                  "end": 2431
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2420,
                                "end": 2431
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2444,
                                  "end": 2448
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2444,
                                "end": 2448
                              }
                            }
                          ],
                          "loc": {
                            "start": 2370,
                            "end": 2458
                          }
                        },
                        "loc": {
                          "start": 2357,
                          "end": 2458
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 2467,
                            "end": 2476
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
                                  "start": 2491,
                                  "end": 2493
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2491,
                                "end": 2493
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2506,
                                  "end": 2511
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2506,
                                "end": 2511
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
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
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 2541,
                                  "end": 2548
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2541,
                                "end": 2548
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 2561,
                                  "end": 2573
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
                                        "start": 2592,
                                        "end": 2594
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2592,
                                      "end": 2594
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 2611,
                                        "end": 2619
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2611,
                                      "end": 2619
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2636,
                                        "end": 2647
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2636,
                                      "end": 2647
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2664,
                                        "end": 2668
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2664,
                                      "end": 2668
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2574,
                                  "end": 2682
                                }
                              },
                              "loc": {
                                "start": 2561,
                                "end": 2682
                              }
                            }
                          ],
                          "loc": {
                            "start": 2477,
                            "end": 2692
                          }
                        },
                        "loc": {
                          "start": 2467,
                          "end": 2692
                        }
                      }
                    ],
                    "loc": {
                      "start": 2317,
                      "end": 2698
                    }
                  },
                  "loc": {
                    "start": 2304,
                    "end": 2698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2703,
                      "end": 2705
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2703,
                    "end": 2705
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2710,
                      "end": 2714
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2710,
                    "end": 2714
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2719,
                      "end": 2730
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2719,
                    "end": 2730
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2735,
                      "end": 2738
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
                            "start": 2749,
                            "end": 2758
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2749,
                          "end": 2758
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2767,
                            "end": 2774
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2767,
                          "end": 2774
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2783,
                            "end": 2792
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2783,
                          "end": 2792
                        }
                      }
                    ],
                    "loc": {
                      "start": 2739,
                      "end": 2798
                    }
                  },
                  "loc": {
                    "start": 2735,
                    "end": 2798
                  }
                }
              ],
              "loc": {
                "start": 1736,
                "end": 2800
              }
            },
            "loc": {
              "start": 1725,
              "end": 2800
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetings",
              "loc": {
                "start": 2801,
                "end": 2809
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
                      "start": 2816,
                      "end": 2822
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
                          "value": "Label_list",
                          "loc": {
                            "start": 2836,
                            "end": 2846
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2833,
                          "end": 2846
                        }
                      }
                    ],
                    "loc": {
                      "start": 2823,
                      "end": 2852
                    }
                  },
                  "loc": {
                    "start": 2816,
                    "end": 2852
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2857,
                      "end": 2869
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
                            "start": 2880,
                            "end": 2882
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2880,
                          "end": 2882
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2891,
                            "end": 2899
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2891,
                          "end": 2899
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2908,
                            "end": 2919
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2908,
                          "end": 2919
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 2928,
                            "end": 2932
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2928,
                          "end": 2932
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2941,
                            "end": 2945
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2941,
                          "end": 2945
                        }
                      }
                    ],
                    "loc": {
                      "start": 2870,
                      "end": 2951
                    }
                  },
                  "loc": {
                    "start": 2857,
                    "end": 2951
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2956,
                      "end": 2958
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2956,
                    "end": 2958
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2963,
                      "end": 2973
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2963,
                    "end": 2973
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2978,
                      "end": 2988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2978,
                    "end": 2988
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 2993,
                      "end": 3015
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2993,
                    "end": 3015
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnTeamProfile",
                    "loc": {
                      "start": 3020,
                      "end": 3037
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3020,
                    "end": 3037
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "team",
                    "loc": {
                      "start": 3042,
                      "end": 3046
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
                            "start": 3057,
                            "end": 3059
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3057,
                          "end": 3059
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 3068,
                            "end": 3079
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3068,
                          "end": 3079
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 3088,
                            "end": 3094
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3088,
                          "end": 3094
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 3103,
                            "end": 3115
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3103,
                          "end": 3115
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 3124,
                            "end": 3127
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
                                  "start": 3142,
                                  "end": 3155
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3142,
                                "end": 3155
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 3168,
                                  "end": 3177
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3168,
                                "end": 3177
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 3190,
                                  "end": 3201
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3190,
                                "end": 3201
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 3214,
                                  "end": 3223
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3214,
                                "end": 3223
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 3236,
                                  "end": 3245
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3236,
                                "end": 3245
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 3258,
                                  "end": 3265
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3258,
                                "end": 3265
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 3278,
                                  "end": 3290
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3278,
                                "end": 3290
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 3303,
                                  "end": 3311
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3303,
                                "end": 3311
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 3324,
                                  "end": 3338
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
                                        "start": 3357,
                                        "end": 3359
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3357,
                                      "end": 3359
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3376,
                                        "end": 3386
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3376,
                                      "end": 3386
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3403,
                                        "end": 3413
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3403,
                                      "end": 3413
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 3430,
                                        "end": 3437
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3430,
                                      "end": 3437
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3454,
                                        "end": 3465
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3454,
                                      "end": 3465
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3339,
                                  "end": 3479
                                }
                              },
                              "loc": {
                                "start": 3324,
                                "end": 3479
                              }
                            }
                          ],
                          "loc": {
                            "start": 3128,
                            "end": 3489
                          }
                        },
                        "loc": {
                          "start": 3124,
                          "end": 3489
                        }
                      }
                    ],
                    "loc": {
                      "start": 3047,
                      "end": 3495
                    }
                  },
                  "loc": {
                    "start": 3042,
                    "end": 3495
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 3500,
                      "end": 3517
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
                            "start": 3528,
                            "end": 3535
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
                                  "start": 3550,
                                  "end": 3552
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3550,
                                "end": 3552
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3565,
                                  "end": 3575
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3565,
                                "end": 3575
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3588,
                                  "end": 3598
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3588,
                                "end": 3598
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3611,
                                  "end": 3618
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3611,
                                "end": 3618
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3631,
                                  "end": 3642
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3631,
                                "end": 3642
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 3655,
                                  "end": 3660
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
                                        "start": 3679,
                                        "end": 3681
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3679,
                                      "end": 3681
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3698,
                                        "end": 3708
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3698,
                                      "end": 3708
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3725,
                                        "end": 3735
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3725,
                                      "end": 3735
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3752,
                                        "end": 3756
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3752,
                                      "end": 3756
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3773,
                                        "end": 3784
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3773,
                                      "end": 3784
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 3801,
                                        "end": 3813
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3801,
                                      "end": 3813
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "team",
                                      "loc": {
                                        "start": 3830,
                                        "end": 3834
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
                                              "start": 3857,
                                              "end": 3859
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3857,
                                            "end": 3859
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 3880,
                                              "end": 3891
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3880,
                                            "end": 3891
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 3912,
                                              "end": 3918
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3912,
                                            "end": 3918
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 3939,
                                              "end": 3951
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3939,
                                            "end": 3951
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 3972,
                                              "end": 3975
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
                                                    "start": 4002,
                                                    "end": 4015
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4002,
                                                  "end": 4015
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 4040,
                                                    "end": 4049
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4040,
                                                  "end": 4049
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 4074,
                                                    "end": 4085
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4074,
                                                  "end": 4085
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 4110,
                                                    "end": 4119
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4110,
                                                  "end": 4119
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 4144,
                                                    "end": 4153
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4144,
                                                  "end": 4153
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 4178,
                                                    "end": 4185
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4178,
                                                  "end": 4185
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 4210,
                                                    "end": 4222
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4210,
                                                  "end": 4222
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 4247,
                                                    "end": 4255
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4247,
                                                  "end": 4255
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 4280,
                                                    "end": 4294
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
                                                          "start": 4325,
                                                          "end": 4327
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4325,
                                                        "end": 4327
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 4356,
                                                          "end": 4366
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4356,
                                                        "end": 4366
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 4395,
                                                          "end": 4405
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4395,
                                                        "end": 4405
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 4434,
                                                          "end": 4441
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4434,
                                                        "end": 4441
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 4470,
                                                          "end": 4481
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4470,
                                                        "end": 4481
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 4295,
                                                    "end": 4507
                                                  }
                                                },
                                                "loc": {
                                                  "start": 4280,
                                                  "end": 4507
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3976,
                                              "end": 4529
                                            }
                                          },
                                          "loc": {
                                            "start": 3972,
                                            "end": 4529
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3835,
                                        "end": 4547
                                      }
                                    },
                                    "loc": {
                                      "start": 3830,
                                      "end": 4547
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4564,
                                        "end": 4576
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
                                              "start": 4599,
                                              "end": 4601
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4599,
                                            "end": 4601
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4622,
                                              "end": 4630
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4622,
                                            "end": 4630
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4651,
                                              "end": 4662
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4651,
                                            "end": 4662
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4577,
                                        "end": 4680
                                      }
                                    },
                                    "loc": {
                                      "start": 4564,
                                      "end": 4680
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3661,
                                  "end": 4694
                                }
                              },
                              "loc": {
                                "start": 3655,
                                "end": 4694
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 4707,
                                  "end": 4710
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
                                        "start": 4729,
                                        "end": 4738
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4729,
                                      "end": 4738
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 4755,
                                        "end": 4764
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4755,
                                      "end": 4764
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4711,
                                  "end": 4778
                                }
                              },
                              "loc": {
                                "start": 4707,
                                "end": 4778
                              }
                            }
                          ],
                          "loc": {
                            "start": 3536,
                            "end": 4788
                          }
                        },
                        "loc": {
                          "start": 3528,
                          "end": 4788
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4797,
                            "end": 4799
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4797,
                          "end": 4799
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4808,
                            "end": 4818
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4808,
                          "end": 4818
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4827,
                            "end": 4837
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4827,
                          "end": 4837
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4846,
                            "end": 4850
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4846,
                          "end": 4850
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 4859,
                            "end": 4870
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4859,
                          "end": 4870
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 4879,
                            "end": 4891
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4879,
                          "end": 4891
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "team",
                          "loc": {
                            "start": 4900,
                            "end": 4904
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
                                  "start": 4919,
                                  "end": 4921
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4919,
                                "end": 4921
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 4934,
                                  "end": 4945
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4934,
                                "end": 4945
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 4958,
                                  "end": 4964
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4958,
                                "end": 4964
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 4977,
                                  "end": 4989
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4977,
                                "end": 4989
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5002,
                                  "end": 5005
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
                                        "start": 5024,
                                        "end": 5037
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5024,
                                      "end": 5037
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 5054,
                                        "end": 5063
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5054,
                                      "end": 5063
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 5080,
                                        "end": 5091
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5080,
                                      "end": 5091
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 5108,
                                        "end": 5117
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5108,
                                      "end": 5117
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 5134,
                                        "end": 5143
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5134,
                                      "end": 5143
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 5160,
                                        "end": 5167
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5160,
                                      "end": 5167
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 5184,
                                        "end": 5196
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5184,
                                      "end": 5196
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 5213,
                                        "end": 5221
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5213,
                                      "end": 5221
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 5238,
                                        "end": 5252
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
                                              "start": 5275,
                                              "end": 5277
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5275,
                                            "end": 5277
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5298,
                                              "end": 5308
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5298,
                                            "end": 5308
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 5329,
                                              "end": 5339
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5329,
                                            "end": 5339
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 5360,
                                              "end": 5367
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5360,
                                            "end": 5367
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 5388,
                                              "end": 5399
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5388,
                                            "end": 5399
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5253,
                                        "end": 5417
                                      }
                                    },
                                    "loc": {
                                      "start": 5238,
                                      "end": 5417
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5006,
                                  "end": 5431
                                }
                              },
                              "loc": {
                                "start": 5002,
                                "end": 5431
                              }
                            }
                          ],
                          "loc": {
                            "start": 4905,
                            "end": 5441
                          }
                        },
                        "loc": {
                          "start": 4900,
                          "end": 5441
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5450,
                            "end": 5462
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
                                  "start": 5477,
                                  "end": 5479
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5477,
                                "end": 5479
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5492,
                                  "end": 5500
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5492,
                                "end": 5500
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5513,
                                  "end": 5524
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5513,
                                "end": 5524
                              }
                            }
                          ],
                          "loc": {
                            "start": 5463,
                            "end": 5534
                          }
                        },
                        "loc": {
                          "start": 5450,
                          "end": 5534
                        }
                      }
                    ],
                    "loc": {
                      "start": 3518,
                      "end": 5540
                    }
                  },
                  "loc": {
                    "start": 3500,
                    "end": 5540
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 5545,
                      "end": 5559
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5545,
                    "end": 5559
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 5564,
                      "end": 5576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5564,
                    "end": 5576
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5581,
                      "end": 5584
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
                            "start": 5595,
                            "end": 5604
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5595,
                          "end": 5604
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 5613,
                            "end": 5622
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5613,
                          "end": 5622
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5631,
                            "end": 5640
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5631,
                          "end": 5640
                        }
                      }
                    ],
                    "loc": {
                      "start": 5585,
                      "end": 5646
                    }
                  },
                  "loc": {
                    "start": 5581,
                    "end": 5646
                  }
                }
              ],
              "loc": {
                "start": 2810,
                "end": 5648
              }
            },
            "loc": {
              "start": 2801,
              "end": 5648
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 5649,
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
                    "value": "projectVersion",
                    "loc": {
                      "start": 5667,
                      "end": 5681
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
                            "start": 5692,
                            "end": 5694
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5692,
                          "end": 5694
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 5703,
                            "end": 5713
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5703,
                          "end": 5713
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 5722,
                            "end": 5730
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5722,
                          "end": 5730
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5739,
                            "end": 5748
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5739,
                          "end": 5748
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 5757,
                            "end": 5769
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5757,
                          "end": 5769
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 5778,
                            "end": 5790
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5778,
                          "end": 5790
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 5799,
                            "end": 5803
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
                                  "start": 5818,
                                  "end": 5820
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5818,
                                "end": 5820
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 5833,
                                  "end": 5842
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5833,
                                "end": 5842
                              }
                            }
                          ],
                          "loc": {
                            "start": 5804,
                            "end": 5852
                          }
                        },
                        "loc": {
                          "start": 5799,
                          "end": 5852
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5861,
                            "end": 5873
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
                                  "start": 5888,
                                  "end": 5890
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5888,
                                "end": 5890
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5903,
                                  "end": 5911
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5903,
                                "end": 5911
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5924,
                                  "end": 5935
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5924,
                                "end": 5935
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 5948,
                                  "end": 5952
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5948,
                                "end": 5952
                              }
                            }
                          ],
                          "loc": {
                            "start": 5874,
                            "end": 5962
                          }
                        },
                        "loc": {
                          "start": 5861,
                          "end": 5962
                        }
                      }
                    ],
                    "loc": {
                      "start": 5682,
                      "end": 5968
                    }
                  },
                  "loc": {
                    "start": 5667,
                    "end": 5968
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5973,
                      "end": 5975
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5973,
                    "end": 5975
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5980,
                      "end": 5989
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5980,
                    "end": 5989
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 5994,
                      "end": 6013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5994,
                    "end": 6013
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6018,
                      "end": 6033
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6018,
                    "end": 6033
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 6038,
                      "end": 6047
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6038,
                    "end": 6047
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6052,
                      "end": 6063
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6052,
                    "end": 6063
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 6068,
                      "end": 6079
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6068,
                    "end": 6079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6084,
                      "end": 6088
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6084,
                    "end": 6088
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 6093,
                      "end": 6099
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6093,
                    "end": 6099
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 6104,
                      "end": 6114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6104,
                    "end": 6114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "team",
                    "loc": {
                      "start": 6119,
                      "end": 6123
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
                          "value": "Team_nav",
                          "loc": {
                            "start": 6137,
                            "end": 6145
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6134,
                          "end": 6145
                        }
                      }
                    ],
                    "loc": {
                      "start": 6124,
                      "end": 6151
                    }
                  },
                  "loc": {
                    "start": 6119,
                    "end": 6151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 6156,
                      "end": 6160
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
                          "value": "User_nav",
                          "loc": {
                            "start": 6174,
                            "end": 6182
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6171,
                          "end": 6182
                        }
                      }
                    ],
                    "loc": {
                      "start": 6161,
                      "end": 6188
                    }
                  },
                  "loc": {
                    "start": 6156,
                    "end": 6188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6193,
                      "end": 6196
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
                            "start": 6207,
                            "end": 6216
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6207,
                          "end": 6216
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6225,
                            "end": 6234
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6225,
                          "end": 6234
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6243,
                            "end": 6250
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6243,
                          "end": 6250
                        }
                      }
                    ],
                    "loc": {
                      "start": 6197,
                      "end": 6256
                    }
                  },
                  "loc": {
                    "start": 6193,
                    "end": 6256
                  }
                }
              ],
              "loc": {
                "start": 5661,
                "end": 6258
              }
            },
            "loc": {
              "start": 5649,
              "end": 6258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 6259,
                "end": 6270
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
                    "value": "routineVersion",
                    "loc": {
                      "start": 6277,
                      "end": 6291
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
                            "start": 6302,
                            "end": 6304
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6302,
                          "end": 6304
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 6313,
                            "end": 6323
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6313,
                          "end": 6323
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 6332,
                            "end": 6345
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6332,
                          "end": 6345
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 6354,
                            "end": 6364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6354,
                          "end": 6364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 6373,
                            "end": 6382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6373,
                          "end": 6382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 6391,
                            "end": 6399
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6391,
                          "end": 6399
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6408,
                            "end": 6417
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6408,
                          "end": 6417
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6426,
                            "end": 6430
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
                                  "start": 6445,
                                  "end": 6447
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6445,
                                "end": 6447
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 6460,
                                  "end": 6470
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6460,
                                "end": 6470
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6483,
                                  "end": 6492
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6483,
                                "end": 6492
                              }
                            }
                          ],
                          "loc": {
                            "start": 6431,
                            "end": 6502
                          }
                        },
                        "loc": {
                          "start": 6426,
                          "end": 6502
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6511,
                            "end": 6523
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
                                  "start": 6538,
                                  "end": 6540
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6538,
                                "end": 6540
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6553,
                                  "end": 6561
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6553,
                                "end": 6561
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6574,
                                  "end": 6585
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6574,
                                "end": 6585
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 6598,
                                  "end": 6610
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6598,
                                "end": 6610
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 6623,
                                  "end": 6627
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6623,
                                "end": 6627
                              }
                            }
                          ],
                          "loc": {
                            "start": 6524,
                            "end": 6637
                          }
                        },
                        "loc": {
                          "start": 6511,
                          "end": 6637
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6646,
                            "end": 6658
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6646,
                          "end": 6658
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6667,
                            "end": 6679
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6667,
                          "end": 6679
                        }
                      }
                    ],
                    "loc": {
                      "start": 6292,
                      "end": 6685
                    }
                  },
                  "loc": {
                    "start": 6277,
                    "end": 6685
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6690,
                      "end": 6692
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6690,
                    "end": 6692
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6697,
                      "end": 6706
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6697,
                    "end": 6706
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6711,
                      "end": 6730
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6711,
                    "end": 6730
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6735,
                      "end": 6750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6735,
                    "end": 6750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 6755,
                      "end": 6764
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6755,
                    "end": 6764
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6769,
                      "end": 6780
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6769,
                    "end": 6780
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 6785,
                      "end": 6796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6785,
                    "end": 6796
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6801,
                      "end": 6805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6801,
                    "end": 6805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 6810,
                      "end": 6816
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6810,
                    "end": 6816
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 6821,
                      "end": 6831
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6821,
                    "end": 6831
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 6836,
                      "end": 6847
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6836,
                    "end": 6847
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 6852,
                      "end": 6871
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6852,
                    "end": 6871
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "team",
                    "loc": {
                      "start": 6876,
                      "end": 6880
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
                          "value": "Team_nav",
                          "loc": {
                            "start": 6894,
                            "end": 6902
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6891,
                          "end": 6902
                        }
                      }
                    ],
                    "loc": {
                      "start": 6881,
                      "end": 6908
                    }
                  },
                  "loc": {
                    "start": 6876,
                    "end": 6908
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 6913,
                      "end": 6917
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
                          "value": "User_nav",
                          "loc": {
                            "start": 6931,
                            "end": 6939
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6928,
                          "end": 6939
                        }
                      }
                    ],
                    "loc": {
                      "start": 6918,
                      "end": 6945
                    }
                  },
                  "loc": {
                    "start": 6913,
                    "end": 6945
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6950,
                      "end": 6953
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
                            "start": 6964,
                            "end": 6973
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6964,
                          "end": 6973
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6982,
                            "end": 6991
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6982,
                          "end": 6991
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7000,
                            "end": 7007
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7000,
                          "end": 7007
                        }
                      }
                    ],
                    "loc": {
                      "start": 6954,
                      "end": 7013
                    }
                  },
                  "loc": {
                    "start": 6950,
                    "end": 7013
                  }
                }
              ],
              "loc": {
                "start": 6271,
                "end": 7015
              }
            },
            "loc": {
              "start": 6259,
              "end": 7015
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7016,
                "end": 7018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7016,
              "end": 7018
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7019,
                "end": 7029
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7019,
              "end": 7029
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7030,
                "end": 7040
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7030,
              "end": 7040
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 7041,
                "end": 7050
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7041,
              "end": 7050
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 7051,
                "end": 7058
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7051,
              "end": 7058
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 7059,
                "end": 7067
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7059,
              "end": 7067
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 7068,
                "end": 7078
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
                      "start": 7085,
                      "end": 7087
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7085,
                    "end": 7087
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 7092,
                      "end": 7109
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7092,
                    "end": 7109
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 7114,
                      "end": 7126
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7114,
                    "end": 7126
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 7131,
                      "end": 7141
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7131,
                    "end": 7141
                  }
                }
              ],
              "loc": {
                "start": 7079,
                "end": 7143
              }
            },
            "loc": {
              "start": 7068,
              "end": 7143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 7144,
                "end": 7155
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
                      "start": 7162,
                      "end": 7164
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7162,
                    "end": 7164
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 7169,
                      "end": 7183
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7169,
                    "end": 7183
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 7188,
                      "end": 7196
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7188,
                    "end": 7196
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 7201,
                      "end": 7210
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7201,
                    "end": 7210
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 7215,
                      "end": 7225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7215,
                    "end": 7225
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 7230,
                      "end": 7235
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7230,
                    "end": 7235
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 7240,
                      "end": 7247
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7240,
                    "end": 7247
                  }
                }
              ],
              "loc": {
                "start": 7156,
                "end": 7249
              }
            },
            "loc": {
              "start": 7144,
              "end": 7249
            }
          }
        ],
        "loc": {
          "start": 1694,
          "end": 7251
        }
      },
      "loc": {
        "start": 1659,
        "end": 7251
      }
    },
    "Team_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Team_nav",
        "loc": {
          "start": 7261,
          "end": 7269
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Team",
          "loc": {
            "start": 7273,
            "end": 7277
          }
        },
        "loc": {
          "start": 7273,
          "end": 7277
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
                "start": 7280,
                "end": 7282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7280,
              "end": 7282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7283,
                "end": 7294
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7283,
              "end": 7294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7295,
                "end": 7301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7295,
              "end": 7301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7302,
                "end": 7314
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7302,
              "end": 7314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7315,
                "end": 7318
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
                      "start": 7325,
                      "end": 7338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7325,
                    "end": 7338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7343,
                      "end": 7352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7343,
                    "end": 7352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7357,
                      "end": 7368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7357,
                    "end": 7368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7373,
                      "end": 7382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7373,
                    "end": 7382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7387,
                      "end": 7396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7387,
                    "end": 7396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7401,
                      "end": 7408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7401,
                    "end": 7408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7413,
                      "end": 7425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7413,
                    "end": 7425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7430,
                      "end": 7438
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7430,
                    "end": 7438
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 7443,
                      "end": 7457
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
                            "start": 7468,
                            "end": 7470
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7468,
                          "end": 7470
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7479,
                            "end": 7489
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7479,
                          "end": 7489
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 7498,
                            "end": 7508
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7498,
                          "end": 7508
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 7517,
                            "end": 7524
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7517,
                          "end": 7524
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 7533,
                            "end": 7544
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7533,
                          "end": 7544
                        }
                      }
                    ],
                    "loc": {
                      "start": 7458,
                      "end": 7550
                    }
                  },
                  "loc": {
                    "start": 7443,
                    "end": 7550
                  }
                }
              ],
              "loc": {
                "start": 7319,
                "end": 7552
              }
            },
            "loc": {
              "start": 7315,
              "end": 7552
            }
          }
        ],
        "loc": {
          "start": 7278,
          "end": 7554
        }
      },
      "loc": {
        "start": 7252,
        "end": 7554
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7564,
          "end": 7572
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7576,
            "end": 7580
          }
        },
        "loc": {
          "start": 7576,
          "end": 7580
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
                "start": 7583,
                "end": 7585
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7583,
              "end": 7585
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7586,
                "end": 7596
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7586,
              "end": 7596
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7597,
                "end": 7607
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7597,
              "end": 7607
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7608,
                "end": 7619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7608,
              "end": 7619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7620,
                "end": 7626
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7620,
              "end": 7626
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7627,
                "end": 7632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7627,
              "end": 7632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 7633,
                "end": 7653
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7633,
              "end": 7653
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7654,
                "end": 7658
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7654,
              "end": 7658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7659,
                "end": 7671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7659,
              "end": 7671
            }
          }
        ],
        "loc": {
          "start": 7581,
          "end": 7673
        }
      },
      "loc": {
        "start": 7555,
        "end": 7673
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "home",
      "loc": {
        "start": 7681,
        "end": 7685
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
              "start": 7687,
              "end": 7692
            }
          },
          "loc": {
            "start": 7686,
            "end": 7692
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "HomeInput",
              "loc": {
                "start": 7694,
                "end": 7703
              }
            },
            "loc": {
              "start": 7694,
              "end": 7703
            }
          },
          "loc": {
            "start": 7694,
            "end": 7704
          }
        },
        "directives": [],
        "loc": {
          "start": 7686,
          "end": 7704
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
            "value": "home",
            "loc": {
              "start": 7710,
              "end": 7714
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7715,
                  "end": 7720
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7723,
                    "end": 7728
                  }
                },
                "loc": {
                  "start": 7722,
                  "end": 7728
                }
              },
              "loc": {
                "start": 7715,
                "end": 7728
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
                  "value": "recommended",
                  "loc": {
                    "start": 7736,
                    "end": 7747
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
                        "value": "Resource_list",
                        "loc": {
                          "start": 7761,
                          "end": 7774
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7758,
                        "end": 7774
                      }
                    }
                  ],
                  "loc": {
                    "start": 7748,
                    "end": 7780
                  }
                },
                "loc": {
                  "start": 7736,
                  "end": 7780
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 7785,
                    "end": 7794
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
                        "value": "Reminder_full",
                        "loc": {
                          "start": 7808,
                          "end": 7821
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7805,
                        "end": 7821
                      }
                    }
                  ],
                  "loc": {
                    "start": 7795,
                    "end": 7827
                  }
                },
                "loc": {
                  "start": 7785,
                  "end": 7827
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 7832,
                    "end": 7841
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
                        "value": "Resource_list",
                        "loc": {
                          "start": 7855,
                          "end": 7868
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7852,
                        "end": 7868
                      }
                    }
                  ],
                  "loc": {
                    "start": 7842,
                    "end": 7874
                  }
                },
                "loc": {
                  "start": 7832,
                  "end": 7874
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 7879,
                    "end": 7888
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
                        "value": "Schedule_list",
                        "loc": {
                          "start": 7902,
                          "end": 7915
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7899,
                        "end": 7915
                      }
                    }
                  ],
                  "loc": {
                    "start": 7889,
                    "end": 7921
                  }
                },
                "loc": {
                  "start": 7879,
                  "end": 7921
                }
              }
            ],
            "loc": {
              "start": 7730,
              "end": 7925
            }
          },
          "loc": {
            "start": 7710,
            "end": 7925
          }
        }
      ],
      "loc": {
        "start": 7706,
        "end": 7927
      }
    },
    "loc": {
      "start": 7675,
      "end": 7927
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
